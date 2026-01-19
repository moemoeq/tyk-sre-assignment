package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	apiV1 := v1.New(cfg, kClient)
	srv := server.New(ctx, *address, apiV1)

	// Start Server in a separate goroutine
	// for graceful shutdown
	go func() {
		fmt.Printf("Server listening on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt
	<-ctx.Done()

	// Shutdown gracefully
	fmt.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.GracefulTimeout)*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	fmt.Println("Server exiting")
}
