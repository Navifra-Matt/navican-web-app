package main

import (
	"can-db-writer/internal/api"
	"can-db-writer/internal/config"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Command line flag for config file
	envFile := flag.String("env", ".env", "Path to .env configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting CAN Database API Server...")
	log.Printf("HTTP Server Port: %d", cfg.APIPort)
	log.Printf("gRPC Server Port: %d", cfg.GRPCPort)
	log.Printf("ClickHouse: %s:%d/%s.%s", cfg.ClickHouseHost, cfg.ClickHousePort, cfg.ClickHouseDatabase, cfg.ClickHouseTable)
	log.Printf("InfluxDB: %s/%s", cfg.InfluxDBURL, cfg.InfluxDBDatabase)

	// Create API server configuration
	serverConfig := api.ServerConfig{
		Port:             cfg.APIPort,
		GRPCPort:         cfg.GRPCPort,
		CHHost:           cfg.ClickHouseHost,
		CHPort:           cfg.ClickHousePort,
		CHDatabase:       cfg.ClickHouseDatabase,
		CHUsername:       cfg.ClickHouseUsername,
		CHPassword:       cfg.ClickHousePassword,
		CHTable:          cfg.ClickHouseTable,
		CHStatsTable:     cfg.ClickHouseStatsTable,
		InfluxDBURL:      cfg.InfluxDBURL,
		InfluxDBToken:    cfg.InfluxDBToken,
		InfluxDBDatabase: cfg.InfluxDBDatabase,
	}

	// Create and start API server
	server, err := api.NewServer(serverConfig)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("API Server started successfully")
	log.Printf("HTTP API available at: http://localhost:%d/", cfg.APIPort)
	log.Printf("gRPC API available at: localhost:%d", cfg.GRPCPort)
	log.Println("Press Ctrl+C to stop")

	// Wait for termination signal
	<-sigChan
	log.Println("\nShutting down API server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("API server stopped")
}
