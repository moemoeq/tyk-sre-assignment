package main

import (
	"flag"
	"fmt"

	v1 "github.com/moemoeq/tyk-sre-app/internal/api/v1"
	"github.com/moemoeq/tyk-sre-app/internal/config"
	"github.com/moemoeq/tyk-sre-app/internal/k8s"
	"github.com/moemoeq/tyk-sre-app/internal/server"
)

func main() {
	cfg := config.Load()
	fmt.Printf("APP Environment: %s\n", cfg.Environment)

	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig, leave empty for in-cluster")
	address := flag.String("address", ":"+cfg.Port, "HTTP server listen address")
	flag.Parse()

	kClient, err := k8s.NewClient(*kubeconfig)
	if err != nil {
		panic(err)
	}

	version, err := k8s.GetKubernetesVersion(kClient.Clientset)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Connected to Kubernetes %s\n", version)

	// Initialize API Server
	apiV1 := v1.New(cfg, kClient)
	srv := server.New(*address, apiV1)

	// Start Server
	fmt.Printf("Server listening on %s\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
