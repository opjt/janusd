package main

import (
	"log/slog"
	"os"

	k8sstore "karden/internal/store/k8s"
	"karden/internal/watcher"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// structured logging to stdout
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfig for local development
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

	store := k8sstore.New(clientset)
	w := watcher.New(clientset, store)

	w.Start()
}
