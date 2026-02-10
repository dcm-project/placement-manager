package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dcm-project/placement-manager/internal/apiserver"
	"github.com/dcm-project/placement-manager/internal/config"
	"github.com/dcm-project/placement-manager/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create TCP listener
	listener, err := net.Listen("tcp", cfg.Service.Address)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	// Initialize handler
	handler := handlers.NewHandler()

	// Create API server
	srv := apiserver.New(cfg, listener, handler)

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("Starting Placement Manager API server on %s", listener.Addr().String())
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
