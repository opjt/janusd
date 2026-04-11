package watcher

import (
	"context"
	"log/slog"
	"strconv"

	"karden/internal/domain/workload"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	client kubernetes.Interface
	store  workload.SecretStore
	repo   workload.Repository
	stopCh chan struct{}
}

func New(client kubernetes.Interface, store workload.SecretStore, repo workload.Repository) *Watcher {
	return &Watcher{
		client: client,
		store:  store,
		repo:   repo,
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

	slog.Info("watcher stopped")
}

func (w *Watcher) Stop() {
	close(w.stopCh)
}

func (w *Watcher) handlePod(pod *corev1.Pod) {
	if pod.Annotations[AnnotationInject] != "true" {
		return
	}

	t := parseTarget(pod)
	if t == nil {
		return
	}

	slog.Info("detected managed pod",
		"namespace", pod.Namespace,
		"pod", pod.Name,
		"secret", t.SecretName,
	)

	ctx := context.Background()
	w.ensureSecret(ctx, t)
	w.upsertWorkload(ctx, t)
}

// parseTarget extracts a ManagedWorkload from pod annotations.
// Returns nil if required annotations are missing.
func parseTarget(pod *corev1.Pod) *workload.ManagedWorkload {
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

	dbPort := defaultDBPort(workload.DBType(ann[AnnotationDBType]))
	if v := ann[AnnotationDBPort]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			dbPort = n
		}
	}

	return &workload.ManagedWorkload{
		PodName:      pod.Name,
		Namespace:    pod.Namespace,
		SecretName:   secretName,
		Type:         workload.Type(ann[AnnotationType]),
		DBType:       workload.DBType(ann[AnnotationDBType]),
		DBHost:       ann[AnnotationDBHost],
		DBPort:       dbPort,
		RotationDays: rotationDays,
		Status:       workload.StatusActive,
	}
}

func (w *Watcher) upsertWorkload(ctx context.Context, t *workload.ManagedWorkload) {
	if err := w.repo.Upsert(ctx, t); err != nil {
		slog.Error("failed to upsert workload",
			"pod", t.PodName,
			"namespace", t.Namespace,
			"err", err,
		)
		return
	}
	slog.Info("workload upserted",
		"pod", t.PodName,
		"namespace", t.Namespace,
	)
}

// ensureSecret creates the Secret if it doesn't exist yet.
func (w *Watcher) ensureSecret(ctx context.Context, t *workload.ManagedWorkload) {
	existing, err := w.store.Get(ctx, t.Namespace, t.SecretName, "")
	if err == nil && existing != "" {
		slog.Info("secret already exists, skipping",
			"secret", t.SecretName,
		)
		return
	}

	data := buildSecretData(t)
	if err := w.store.Set(ctx, t.Namespace, t.SecretName, data); err != nil {
		slog.Error("failed to create secret",
			"secret", t.SecretName,
			"err", err,
		)
		return
	}

	slog.Info("secret created",
		"secret", t.SecretName,
		"namespace", t.Namespace,
	)
}

// buildSecretData generates initial secret values based on type.
func buildSecretData(t *workload.ManagedWorkload) map[string]string {
	switch t.Type {
	case workload.TypeDatabase:
		return buildDBSecretData(t)
	default:
		return map[string]string{}
	}
}

func buildDBSecretData(t *workload.ManagedWorkload) map[string]string {
	username := buildUsername(t.SecretName)
	password := generatePassword()

	switch t.DBType {
	case workload.DBTypePostgres:
		return map[string]string{
			"POSTGRES_USER":     username,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       "app",
		}
	case workload.DBTypeMySQL:
		return map[string]string{
			"MYSQL_USER":          username,
			"MYSQL_PASSWORD":      password,
			"MYSQL_ROOT_PASSWORD": generatePassword(),
		}
	default:
		return map[string]string{
			"USERNAME": username,
			"PASSWORD": password,
		}
	}
}
