package api

import (
	pb "can-db-writer/internal/proto/can"
	"fmt"
	"log"
	"net"

	cangrpc "can-db-writer/internal/grpc"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer wraps the gRPC server
type GRPCServer struct {
	server     *grpc.Server
	listener   net.Listener
	canService *cangrpc.CANServer
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(port int, chConn driver.Conn, tableName string) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	canService := cangrpc.NewCANServer(chConn, tableName)

	// Register the service
	pb.RegisterCanServiceServer(grpcServer, canService)

	// Register reflection service for tools like grpcurl
	reflection.Register(grpcServer)

	return &GRPCServer{
		server:     grpcServer,
		listener:   lis,
		canService: canService,
	}, nil
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	log.Printf("gRPC server listening on %s", s.listener.Addr().String())
	return s.server.Serve(s.listener)
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop() {
	log.Println("Stopping gRPC server...")
	s.server.GracefulStop()
}
