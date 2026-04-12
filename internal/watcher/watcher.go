package watcher

import (
	"context"
	"log/slog"
	"strconv"

	"karden/internal/domain/audit"
	"karden/internal/domain/workload"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	client    kubernetes.Interface
	store     workload.SecretStore
	repo      workload.Repository
	auditRepo audit.Repository
	stopCh    chan struct{}
}

func New(client kubernetes.Interface, store workload.SecretStore, repo workload.Repository, auditRepo audit.Repository) *Watcher {
	return &Watcher{
		client:    client,
		store:     store,
		repo:      repo,
		auditRepo: auditRepo,
		stopCh:    make(chan struct{}),
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
		DeleteFunc: func(obj any) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				// handle tombstone object from the informer cache
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				pod, ok = tombstone.Obj.(*corev1.Pod)
				if !ok {
					return
				}
			}
			w.handlePodDeleted(pod)
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
	id := w.upsertWorkload(ctx, t)
	if id > 0 {
		w.ensureSecret(ctx, t, id)
	}
}

func (w *Watcher) handlePodDeleted(pod *corev1.Pod) {
	if pod.Annotations[AnnotationInject] != "true" {
		return
	}

	ctx := context.Background()
	if err := w.repo.SetInactive(ctx, pod.Name, pod.Namespace); err != nil {
		slog.Error("failed to mark workload inactive",
			"pod", pod.Name,
			"namespace", pod.Namespace,
			"err", err,
		)
		return
	}

	slog.Info("workload marked inactive",
		"pod", pod.Name,
		"namespace", pod.Namespace,
	)
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

// upsertWorkload persists the workload and returns its DB id (0 on error).
func (w *Watcher) upsertWorkload(ctx context.Context, t *workload.ManagedWorkload) int64 {
	id, err := w.repo.Upsert(ctx, t)
	if err != nil {
		slog.Error("failed to upsert workload",
			"pod", t.PodName,
			"namespace", t.Namespace,
			"err", err,
		)
		return 0
	}
	slog.Info("workload upserted",
		"pod", t.PodName,
		"namespace", t.Namespace,
	)
	return id
}

// ensureSecret creates the K8s Secret if it doesn't exist yet, then writes an audit log.
func (w *Watcher) ensureSecret(ctx context.Context, t *workload.ManagedWorkload, workloadID int64) {
	existing, err := w.store.GetData(ctx, t.Namespace, t.SecretName)
	if err == nil && len(existing) > 0 {
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
		_ = w.auditRepo.Save(ctx, &audit.AuditLog{
			TargetID: int(workloadID),
			Action:   audit.ActionCreate,
			Actor:    "karden",
			Result:   audit.ResultFailure,
			Reason:   err.Error(),
		})
		return
	}

	slog.Info("secret created",
		"secret", t.SecretName,
		"namespace", t.Namespace,
	)
	_ = w.auditRepo.Save(ctx, &audit.AuditLog{
		TargetID: int(workloadID),
		Action:   audit.ActionCreate,
		Actor:    "karden",
		Result:   audit.ResultSuccess,
	})
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
