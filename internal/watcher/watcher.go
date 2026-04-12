package watcher

import (
	"context"
	"encoding/json"
	"log/slog"

	"karden/internal/domain/audit"
	"karden/internal/domain/workload"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/informers"
)

var kardenSecretGVR = schema.GroupVersionResource{
	Group:    "karden.io",
	Version:  "v1",
	Resource: "kardensecrets",
}

type Watcher struct {
	dynClient   dynamic.Interface
	client      k8sclient.Interface
	store       workload.SecretStore
	auditRepo   audit.Repository
	factory     dynamicinformer.DynamicSharedInformerFactory
	secretIndex *kardenSecretIndex
	pods        *podMap
	stopCh      chan struct{}
}

func New(dynClient dynamic.Interface, client k8sclient.Interface, store workload.SecretStore, auditRepo audit.Repository) *Watcher {
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynClient, 0)
	ksInformer := factory.ForResource(kardenSecretGVR)

	return &Watcher{
		dynClient:   dynClient,
		client:      client,
		store:       store,
		auditRepo:   auditRepo,
		factory:     factory,
		secretIndex: &kardenSecretIndex{lister: ksInformer.Lister()},
		pods:        newPodMap(),
		stopCh:      make(chan struct{}),
	}
}

// SecretIndex exposes the informer-backed KardenSecret index for the service layer.
func (w *Watcher) SecretIndex() workload.SecretIndex {
	return w.secretIndex
}

// PodIndex exposes the pod-secret index for use by the service layer.
func (w *Watcher) PodIndex() workload.PodIndex {
	return w.pods
}

func (w *Watcher) Start() {
	// KardenSecret informer (factory and lister already initialized in New)
	informer := w.factory.ForResource(kardenSecretGVR).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ks := toKardenSecret(obj)
			if ks != nil {
				w.handleAdd(ks)
			}
		},
		UpdateFunc: func(_, newObj any) {
			ks := toKardenSecret(newObj)
			if ks != nil {
				w.handleAdd(ks)
			}
		},
		DeleteFunc: func(obj any) {
			ks := toKardenSecret(obj)
			if ks == nil {
				// handle tombstone
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				ks = toKardenSecret(tombstone.Obj)
			}
			if ks != nil {
				w.handleDelete(ks)
			}
		},
	})

	// Pod informer — tracks which pods reference which secrets
	podFactory := informers.NewSharedInformerFactory(w.client, 0)
	podInformer := podFactory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if pod, ok := obj.(*corev1.Pod); ok {
				w.pods.set(pod)
			}
		},
		UpdateFunc: func(old, new any) {
			if oldPod, ok := old.(*corev1.Pod); ok {
				w.pods.remove(oldPod)
			}
			if newPod, ok := new.(*corev1.Pod); ok {
				w.pods.set(newPod)
			}
		},
		DeleteFunc: func(obj any) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
					pod, _ = tombstone.Obj.(*corev1.Pod)
				}
			}
			if pod != nil {
				w.pods.remove(pod)
			}
		},
	})

	w.factory.Start(w.stopCh)
	podFactory.Start(w.stopCh)
	w.factory.WaitForCacheSync(w.stopCh)
	podFactory.WaitForCacheSync(w.stopCh)

	slog.Info("watcher started")
	<-w.stopCh
	slog.Info("watcher stopped")
}

func (w *Watcher) Stop() {
	close(w.stopCh)
}

func (w *Watcher) handleAdd(ks *kardenSecret) {
	slog.Info("detected KardenSecret",
		"name", ks.Name,
		"namespace", ks.Namespace,
		"type", ks.Spec.Type,
	)

	ctx := context.Background()
	w.ensureSecret(ctx, ks.toWorkload())
}

func (w *Watcher) handleDelete(ks *kardenSecret) {
	slog.Info("KardenSecret deleted",
		"name", ks.Name,
		"namespace", ks.Namespace,
	)
}

// ensureSecret creates the K8s Secret if it doesn't exist yet, then writes an audit log.
func (w *Watcher) ensureSecret(ctx context.Context, wl *workload.ManagedWorkload) {
	existing, err := w.store.GetData(ctx, wl.Namespace, wl.SecretName)
	if err == nil && len(existing) > 0 {
		slog.Info("secret already exists, skipping", "secret", wl.SecretName)
		return
	}

	data := buildSecretData(wl)
	if err := w.store.Set(ctx, wl.Namespace, wl.SecretName, data); err != nil {
		slog.Error("failed to create secret", "secret", wl.SecretName, "err", err)
		_ = w.auditRepo.Save(ctx, &audit.AuditLog{
			Namespace:  wl.Namespace,
			SecretName: wl.SecretName,
			Action:     audit.ActionCreate,
			Actor:      "karden",
			Result:     audit.ResultFailure,
			Reason:     err.Error(),
		})
		return
	}

	slog.Info("secret created", "secret", wl.SecretName, "namespace", wl.Namespace)
	_ = w.auditRepo.Save(ctx, &audit.AuditLog{
		Namespace:  wl.Namespace,
		SecretName: wl.SecretName,
		Action:     audit.ActionCreate,
		Actor:      "karden",
		Result:     audit.ResultSuccess,
	})
}

// toKardenSecret converts an unstructured K8s object to a KardenSecret.
func toKardenSecret(obj any) *kardenSecret {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}

	specRaw, _, _ := unstructured.NestedMap(u.Object, "spec")
	specBytes, err := json.Marshal(specRaw)
	if err != nil {
		return nil
	}

	var spec kardenSecretSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil
	}

	return &kardenSecret{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
		Spec:      spec,
	}
}
