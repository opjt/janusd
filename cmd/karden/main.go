package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"karden/internal/adapter/k8s"
	"karden/internal/adapter/sqlite"
	"karden/internal/api"
	"karden/internal/domain/workload"
	"karden/internal/pkg/config"
	"karden/internal/watcher"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	env, err := config.NewEnv()
	if err != nil {
		slog.Error("failed to load env", "err", err)
		os.Exit(1)
	}
	if err := run(context.Background(), env); err != nil {
		slog.Error("failed to run", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, env config.Env) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

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
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// SQLite
	db, err := sqlite.Open(env.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	store := k8s.NewSecretStore(clientset)
	repo := sqlite.NewWorkloadRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)

	// watcher start (must be created before service so PodIndex is available)
	w := watcher.New(dynClient, clientset, store, repo, auditRepo)
	secretSvc := workload.NewService(repo, store, w.PodIndex())
	go w.Start()

	// HTTP server
	addr := fmt.Sprintf(":%d", env.Port)

	handler := api.NewHandler(secretSvc, auditRepo)
	srv := api.NewServer(addr, handler)

	errChan := make(chan error, 1)
	go func() {
		slog.Info("http server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		slog.Info("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
		w.Stop()
		slog.Info("server shutdown")
		return nil
	}
}
