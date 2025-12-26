package api

import (
	"can-db-writer/internal/database/clickhouse"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
)

// Server represents the HTTP API server
type Server struct {
	server        *http.Server
	grpcServer    *GRPCServer
	clickhouseAPI *ClickHouseAPI
	statsAPI      *StatsAPI
}

// ServerConfig holds API server configuration
type ServerConfig struct {
	Port             int
	GRPCPort         int
	CHHost           string
	CHPort           int
	CHDatabase       string
	CHUsername       string
	CHPassword       string
	CHTable          string
	CHStatsTable     string
}

// NewServer creates a new API server instance
func NewServer(config ServerConfig) (*Server, error) {
	// Connect to ClickHouse
	chConn, err := clickhousego.Open(&clickhousego.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.CHHost, config.CHPort)},
		Auth: clickhousego.Auth{
			Database: config.CHDatabase,
			Username: config.CHUsername,
			Password: config.CHPassword,
		},
		Settings: clickhousego.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhousego.Compression{
			Method: clickhousego.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test ClickHouse connection
	if err := chConn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// Create ClickHouse writer for export functionality
	chConfig := clickhouse.Config{
		Host:     config.CHHost,
		Port:     config.CHPort,
		Database: config.CHDatabase,
		Username: config.CHUsername,
		Password: config.CHPassword,
		Table:    config.CHTable,
	}
	writer, err := clickhouse.New(chConfig, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClickHouse writer: %w", err)
	}

	// Create API handlers
	clickhouseAPI := NewClickHouseAPI(chConn, config.CHTable, writer)
	statsAPI := NewStatsAPI(chConn, config.CHStatsTable)

	// Create gRPC server if port is specified
	var grpcServer *GRPCServer
	if config.GRPCPort > 0 {
		var err error
		grpcServer, err = NewGRPCServer(config.GRPCPort, chConn, config.CHTable)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC server: %w", err)
		}
	}

	server := &Server{
		clickhouseAPI: clickhouseAPI,
		statsAPI:      statsAPI,
		grpcServer:    grpcServer,
	}

	// Setup HTTP router
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	// Create HTTP server
	server.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      loggingMiddleware(corsMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Root endpoint
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/health", s.handleHealth)

	// ClickHouse endpoints
	mux.HandleFunc("/api/clickhouse/canopen/messages", s.clickhouseAPI.GetCANopenMessages)
	mux.HandleFunc("/api/clickhouse/export", s.clickhouseAPI.ExportData)

	// SocketCAN statistics endpoints
	mux.HandleFunc("/api/stats/latest", s.statsAPI.GetLatestStats)
	mux.HandleFunc("/api/stats/history", s.statsAPI.GetStatsHistory)
	mux.HandleFunc("/api/stats/aggregated", s.statsAPI.GetStatsAggregated)
}

// handleRoot returns API information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	info := map[string]any{
		"name":    "CAN Database API Server",
		"version": "1.0.0",
		"endpoints": map[string]any{
			"health": "/health",
			"clickhouse": map[string]string{
				"messages": "/api/clickhouse/messages?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&can_id=0x123&interface=can0&limit=100&offset=0",
				"count":    "/api/clickhouse/count?start_time=2024-01-01T00:00:00Z&can_id=0x123",
				"can_ids":  "/api/clickhouse/can_ids",
				"stats":    "/api/clickhouse/stats?limit=10",
				"export":   "POST /api/clickhouse/export (body: {start_time, end_time, filename?, compression?}) - Downloads Parquet file",
			},
			"canopen": map[string]string{
				"messages": "/api/clickhouse/canopen/messages?message_type=pdo&start_time=2024-01-01T00:00:00Z&interface=can0&limit=100",
				"stats":    "/api/clickhouse/canopen/stats?start_time=2024-01-01T00:00:00Z&interface=can0",
			},
			"socketcan_stats": map[string]string{
				"latest":     "/api/stats/latest?interface=can0",
				"history":    "/api/stats/history?interface=can0&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&limit=100",
				"aggregated": "/api/stats/aggregated?interface=can0&start_time=2024-01-01T00:00:00Z&interval=1h",
			},
		},
	}

	respondWithJSON(w, http.StatusOK, info)
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now(),
		"services": map[string]string{
			"api":        "up",
			"clickhouse": "connected",
		},
	}

	respondWithJSON(w, http.StatusOK, health)
}

// Start starts the API server
func (s *Server) Start() error {
	// Start gRPC server in background if configured
	if s.grpcServer != nil {
		go func() {
			if err := s.grpcServer.Start(); err != nil {
				log.Printf("gRPC server error: %v", err)
			}
		}()
	}

	log.Printf("Starting HTTP API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping API server...")

	// Stop gRPC server if running
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}

	return s.server.Shutdown(ctx)
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Call next handler
		next.ServeHTTP(w, r)

		// Log duration
		duration := time.Since(start)
		log.Printf("[%s] %s completed in %v", r.Method, r.URL.Path, duration)
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
