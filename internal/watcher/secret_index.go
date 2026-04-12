package watcher

import (
	"karden/internal/domain/workload"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

var _ workload.SecretIndex = (*kardenSecretIndex)(nil)

// kardenSecretIndex implements workload.SecretIndex using the informer's lister cache.
type kardenSecretIndex struct {
	lister cache.GenericLister
}

func (idx *kardenSecretIndex) List(namespace string) []*workload.ManagedWorkload {
	var objs []runtime.Object
	var err error

	if namespace == "" {
		objs, err = idx.lister.List(labels.Everything())
	} else {
		objs, err = idx.lister.ByNamespace(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil
	}

	result := make([]*workload.ManagedWorkload, 0, len(objs))
	for _, obj := range objs {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		ks := toKardenSecret(u)
		if ks != nil {
			result = append(result, ks.toWorkload())
		}
	}
	return result
}

func (idx *kardenSecretIndex) Get(namespace, name string) *workload.ManagedWorkload {
	obj, err := idx.lister.ByNamespace(namespace).Get(name)
	if err != nil {
		return nil
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	ks := toKardenSecret(u)
	if ks == nil {
		return nil
	}
	return ks.toWorkload()
}
