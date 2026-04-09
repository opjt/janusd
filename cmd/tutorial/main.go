package main

import (
	"context"
	"fmt"

	k8sstore "karden/internal/store/k8s"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 1. load kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		panic(err)
	}

	// 2. create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	store := k8sstore.New(clientset)
	ctx := context.Background()

	namespace := "default"
	secretName := "janusd-test-secret"

	// 3. Set — create a Secret
	fmt.Println("--- Set ---")
	err = store.Set(ctx, namespace, secretName, map[string]string{
		"POSTGRES_USER":     "app_user",
		"POSTGRES_PASSWORD": "supersecret123",
		"POSTGRES_DB":       "torchi",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Secret '%s' created\n", secretName)

	// 4. Get — read a value back
	fmt.Println("--- Get ---")
	val, err := store.Get(ctx, namespace, secretName, "POSTGRES_PASSWORD")
	if err != nil {
		panic(err)
	}
	fmt.Printf("POSTGRES_PASSWORD = %s\n", val)

	// 5. Delete — remove the Secret
	fmt.Println("--- Delete ---")
	err = store.Delete(ctx, namespace, secretName)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Secret '%s' deleted\n", secretName)
}
