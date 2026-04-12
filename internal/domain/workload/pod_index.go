package workload

// PodIndex provides a lookup from secret → pods that reference it.
type PodIndex interface {
	GetPods(namespace, secretName string) []string
}
