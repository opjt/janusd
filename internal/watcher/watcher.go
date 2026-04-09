package watcher

import (
	"context"
	"log/slog"
	"strconv"

	"karden/internal/domain"
	"karden/internal/store"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	client kubernetes.Interface
	store  store.SecretStore
	stopCh chan struct{}
}

func New(client kubernetes.Interface, store store.SecretStore) *Watcher {
	return &Watcher{
		client: client,
		store:  store,
		stopCh: make(chan struct{}),
	}
}

func (w *Watcher) Start() {
	factory := informers.NewSharedInformerFactory(w.client, 0)
	podInformer := factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod := obj.(*corev1.Pod)
			w.handlePod(pod)
		},
		UpdateFunc: func(_, newObj any) {
			pod := newObj.(*corev1.Pod)
			w.handlePod(pod)
		},
	})

	factory.Start(w.stopCh)
	factory.WaitForCacheSync(w.stopCh)

	slog.Info("watcher started")
	<-w.stopCh
}

func (w *Watcher) Stop() {
	close(w.stopCh)
}

func (w *Watcher) handlePod(pod *corev1.Pod) {
	if pod.Annotations[AnnotationInject] != "true" {
		return
	}

	target := parseTarget(pod)
	if target == nil {
		return
	}

	slog.Info("detected managed pod",
		"namespace", pod.Namespace,
		"pod", pod.Name,
		"secret", target.SecretName,
	)

	w.ensureSecret(context.Background(), target)
}

// parseTarget extracts a ManagedTarget from pod annotations.
// Returns nil if required annotations are missing.
func parseTarget(pod *corev1.Pod) *domain.ManagedTarget {
	ann := pod.Annotations

	secretName := ann[AnnotationSecretName]
	if secretName == "" {
		slog.Warn("missing annotation",
			"namespace", pod.Namespace,
			"pod", pod.Name,
			"annotation", AnnotationSecretName,
		)
		return nil
	}

	rotationDays := 30
	if v := ann[AnnotationRotationDays]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rotationDays = n
		}
	}

	dbPort := defaultDBPort(domain.DBType(ann[AnnotationDBType]))
	if v := ann[AnnotationDBPort]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			dbPort = n
		}
	}

	return &domain.ManagedTarget{
		PodName:      pod.Name,
		Namespace:    pod.Namespace,
		SecretName:   secretName,
		Type:         domain.Type(ann[AnnotationType]),
		DBType:       domain.DBType(ann[AnnotationDBType]),
		DBHost:       ann[AnnotationDBHost],
		DBPort:       dbPort,
		DBUser:       ann[AnnotationDBUser],
		RotationDays: rotationDays,
		Status:       domain.StatusActive,
	}
}

// ensureSecret creates the Secret if it doesn't exist yet.
func (w *Watcher) ensureSecret(ctx context.Context, target *domain.ManagedTarget) {
	existing, err := w.store.Get(ctx, target.Namespace, target.SecretName, "")
	if err == nil && existing != "" {
		slog.Info("secret already exists, skipping",
			"secret", target.SecretName,
		)
		return
	}

	data := buildSecretData(target)
	if err := w.store.Set(ctx, target.Namespace, target.SecretName, data); err != nil {
		slog.Error("failed to create secret",
			"secret", target.SecretName,
			"err", err,
		)
		return
	}

	slog.Info("secret created",
		"secret", target.SecretName,
		"namespace", target.Namespace,
	)
}

// buildSecretData generates initial secret values based on type.
func buildSecretData(target *domain.ManagedTarget) map[string]string {
	switch target.Type {
	case domain.TypeDatabase:
		return buildDBSecretData(target)
	default:
		return map[string]string{}
	}
}

func buildDBSecretData(target *domain.ManagedTarget) map[string]string {
	password := generatePassword()

	switch target.DBType {
	case domain.DBTypePostgres:
		return map[string]string{
			"POSTGRES_USER":     target.DBUser,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       "app",
		}
	case domain.DBTypeMySQL:
		return map[string]string{
			"MYSQL_USER":          target.DBUser,
			"MYSQL_PASSWORD":      password,
			"MYSQL_ROOT_PASSWORD": generatePassword(),
		}
	default:
		return map[string]string{
			"PASSWORD": password,
		}
	}
}
