package main

import (
	"log/slog"
	"os"

	"karden/internal/adapter/k8s"
	"karden/internal/adapter/sqlite"
	"karden/internal/watcher"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	config, err := rest.InClusterConfig()
	if err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			slog.Error("failed to load kubeconfig", "err", err)
			os.Exit(1)
		}
		slog.Info("running in local mode")
	} else {
		slog.Info("running in cluster mode")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("failed to create clientset", "err", err)
		os.Exit(1)
	}

	// SQLite DB 초기화
	dsn := os.Getenv("KARDEN_DB_PATH")
	if dsn == "" {
		dsn = "karden.db"
	}
	db, err := sqlite.Open(dsn)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	store := k8s.NewSecretStore(clientset)
	repo := sqlite.NewWorkloadRepository(db)
	w := watcher.New(clientset, store, repo)

	w.Start()
}
