package watcher

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

// podMap maintains a thread-safe index of secret → pods that reference it.
// Covers envFrom.secretRef and env[].valueFrom.secretKeyRef.
type podMap struct {
	mu   sync.RWMutex
	data map[string]map[string]struct{} // "namespace/secret" -> set of pod names
}

func newPodMap() *podMap {
	return &podMap{data: make(map[string]map[string]struct{})}
}

func secretKey(namespace, secretName string) string {
	return fmt.Sprintf("%s/%s", namespace, secretName)
}

// GetPods returns the pod names that reference the given secret.
func (m *podMap) GetPods(namespace, secretName string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	set, ok := m.data[secretKey(namespace, secretName)]
	if !ok {
		return []string{}
	}

	pods := make([]string, 0, len(set))
	for pod := range set {
		pods = append(pods, pod)
	}
	return pods
}

func (m *podMap) set(pod *corev1.Pod) {
	refs := referencedSecrets(pod)
	if len(refs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, secretName := range refs {
		k := secretKey(pod.Namespace, secretName)
		if m.data[k] == nil {
			m.data[k] = make(map[string]struct{})
		}
		m.data[k][pod.Name] = struct{}{}
	}
}

func (m *podMap) remove(pod *corev1.Pod) {
	refs := referencedSecrets(pod)
	if len(refs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, secretName := range refs {
		k := secretKey(pod.Namespace, secretName)
		delete(m.data[k], pod.Name)
	}
}

// referencedSecrets collects all secret names referenced by a pod
// via envFrom.secretRef and env[].valueFrom.secretKeyRef.
func referencedSecrets(pod *corev1.Pod) []string {
	seen := map[string]struct{}{}

	for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		// envFrom.secretRef
		for _, src := range c.EnvFrom {
			if src.SecretRef != nil {
				seen[src.SecretRef.Name] = struct{}{}
			}
		}
		// env[].valueFrom.secretKeyRef
		for _, env := range c.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				seen[env.ValueFrom.SecretKeyRef.Name] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}
