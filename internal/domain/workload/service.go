package workload

import (
	"context"
	"time"
)

// SecretView is the secret-centric projection aggregated from ManagedWorkloads
// that share the same secret name and namespace.
type SecretView struct {
	Name          string
	Namespace     string
	Type          Type
	DBType        DBType
	RotationDays  int
	LastRotatedAt *time.Time
	Status        Status
	Pods          []string
	Data          map[string]string // populated only on Get
}

// Service provides secret-centric use cases built on top of workload data.
type Service interface {
	List(ctx context.Context) ([]*SecretView, error)
	Get(ctx context.Context, namespace, name string) (*SecretView, error)
}

type service struct {
	repo     Repository
	store    SecretStore
	podIndex PodIndex
}

func NewService(repo Repository, store SecretStore, podIndex PodIndex) Service {
	return &service{repo: repo, store: store, podIndex: podIndex}
}

func (s *service) List(ctx context.Context) ([]*SecretView, error) {
	workloads, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	type key struct{ name, namespace string }
	index := map[key]*SecretView{}

	for _, wl := range workloads {
		k := key{wl.SecretName, wl.Namespace}
		if _, ok := index[k]; !ok {
			index[k] = &SecretView{
				Name:          wl.SecretName,
				Namespace:     wl.Namespace,
				Type:          wl.Type,
				DBType:        wl.DBType,
				RotationDays:  wl.RotationDays,
				LastRotatedAt: wl.LastRotatedAt,
				Status:        wl.Status,
				Pods:          s.podIndex.GetPods(wl.Namespace, wl.SecretName),
			}
		}
	}

	result := make([]*SecretView, 0, len(index))
	for _, v := range index {
		result = append(result, v)
	}
	return result, nil
}

func (s *service) Get(ctx context.Context, namespace, name string) (*SecretView, error) {
	workloads, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var view *SecretView
	for _, wl := range workloads {
		if wl.SecretName != name || wl.Namespace != namespace {
			continue
		}
		if view == nil {
			view = &SecretView{
				Name:          wl.SecretName,
				Namespace:     wl.Namespace,
				Type:          wl.Type,
				DBType:        wl.DBType,
				RotationDays:  wl.RotationDays,
				LastRotatedAt: wl.LastRotatedAt,
				Status:        wl.Status,
				Pods:          s.podIndex.GetPods(namespace, name),
			}
		}
	}

	if view == nil {
		return nil, nil
	}

	// Fetch actual secret data from the store
	data, err := s.store.GetData(ctx, namespace, name)
	if err == nil {
		view.Data = data
	}

	return view, nil
}
