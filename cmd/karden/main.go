package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"karden/internal/adapter/k8s"
	"karden/internal/adapter/sqlite"
	"karden/internal/api"
	"karden/internal/watcher"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// K8s client
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

	clientset, err := k8sclient.NewForConfig(config)
	if err != nil {
		slog.Error("failed to create clientset", "err", err)
		os.Exit(1)
	}

	// SQLite
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

	// watcher start
	w := watcher.New(clientset, store, repo)
	go w.Start()

	// HTTP server
	addr := os.Getenv("KARDEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	handler := api.NewHandler(repo, store)
	srv := api.NewServer(addr, handler)

	go func() {
		slog.Info("http server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
