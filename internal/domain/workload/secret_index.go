package workload

// SecretIndex provides an in-memory view of all managed KardenSecrets.
type SecretIndex interface {
	List(namespace string) []*ManagedWorkload
	Get(namespace, name string) *ManagedWorkload
}
