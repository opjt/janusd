package workload

import "context"

// SecretView is the secret-centric projection for the API layer.
type SecretView struct {
	Name          string
	Namespace     string
	Type          Type
	DBType        DBType
	RotationDays  int
	LastRotatedAt *string
	Status        Status
	Pods          []string
	Data          map[string]string // populated only on Get
}

// Service provides secret-centric use cases.
type Service interface {
	List(ctx context.Context) ([]*SecretView, error)
	Get(ctx context.Context, namespace, name string) (*SecretView, error)
}

type service struct {
	secrets  SecretIndex
	store    SecretStore
	podIndex PodIndex
}

func NewService(secrets SecretIndex, store SecretStore, podIndex PodIndex) Service {
	return &service{secrets: secrets, store: store, podIndex: podIndex}
}

func (s *service) List(_ context.Context) ([]*SecretView, error) {
	workloads := s.secrets.List("")
	result := make([]*SecretView, 0, len(workloads))
	for _, wl := range workloads {
		result = append(result, s.toView(wl))
	}
	return result, nil
}

func (s *service) Get(ctx context.Context, namespace, name string) (*SecretView, error) {
	wl := s.secrets.Get(namespace, name)
	if wl == nil {
		return nil, nil
	}

	view := s.toView(wl)

	data, err := s.store.GetData(ctx, namespace, name)
	if err == nil {
		view.Data = data
	}

	return view, nil
}

func (s *service) toView(wl *ManagedWorkload) *SecretView {
	var lastRotatedAt *string
	if wl.LastRotatedAt != nil {
		t := wl.LastRotatedAt.UTC().Format("2006-01-02T15:04:05Z")
		lastRotatedAt = &t
	}
	return &SecretView{
		Name:          wl.SecretName,
		Namespace:     wl.Namespace,
		Type:          wl.Type,
		DBType:        wl.DBType,
		RotationDays:  wl.RotationDays,
		LastRotatedAt: lastRotatedAt,
		Status:        wl.Status,
		Pods:          s.podIndex.GetPods(wl.Namespace, wl.SecretName),
	}
}
