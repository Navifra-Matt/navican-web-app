package grpc

import (
	pb "can-db-writer/internal/proto/can"
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CANServer implements the gRPC canService
type CANServer struct {
	pb.UnimplementedCanServiceServer
	conn      driver.Conn
	tableName string
}

// NewCANServer creates a new gRPC CAN server
func NewCANServer(conn driver.Conn, tableName string) *CANServer {
	return &CANServer{
		conn:      conn,
		tableName: tableName,
	}
}

// GetCANopenMessages retrieves CANopen messages classified by message type
func (s *CANServer) GetCANopenMessages(ctx context.Context, req *pb.GetCANopenMessagesRequest) (*pb.GetCANopenMessagesResponse, error) {
	query := fmt.Sprintf("SELECT timestamp, interface, can_id, hex(can_id) as can_id_hex, data FROM %s WHERE 1=1", s.tableName)
	args := make([]any, 0)

	if req.Filter != nil {
		if req.Filter.StartTime != nil {
			query += " AND timestamp >= ?"
			args = append(args, req.Filter.StartTime.AsTime())
		}
		if req.Filter.EndTime != nil {
			query += " AND timestamp <= ?"
			args = append(args, req.Filter.EndTime.AsTime())
		}
		if req.Filter.Interface != "" {
			query += " AND interface = ?"
			args = append(args, req.Filter.Interface)
		}
	}

	query += " ORDER BY timestamp DESC"

	if req.Filter != nil && req.Filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Filter.Limit)
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*pb.CANopenMessage
	for rows.Next() {
		var ts time.Time
		var iface string
		var canID uint32
		var canIDHex string
		var data []byte

		if err := rows.Scan(&ts, &iface, &canID, &canIDHex, &data); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		msgType, nodeID := classifyCANopenMessage(canID)

		// Filter by message type if specified
		if req.MessageType != "" && msgType != req.MessageType {
			continue
		}

		// Filter by node ID if specified
		if req.NodeId != nil && nodeID != *req.NodeId {
			continue
		}

		canopenMsg := &pb.CANopenMessage{
			Timestamp:   timestamppb.New(ts),
			Interface:   iface,
			CanId:       canID,
			CanIdHex:    canIDHex,
			Data:        data,
			MessageType: msgType,
			NodeId:      nodeID,
			ParsedData:  make(map[string]string),
		}

		// Parse PDO data if mappings are provided
		if (msgType == "tpdo" || msgType == "rpdo") && len(req.PdoMappings) > 0 {
			// PDO parsing logic would go here
			// For now, just return empty parsed data
		}

		messages = append(messages, canopenMsg)
	}

	return &pb.GetCANopenMessagesResponse{Messages: messages}, nil
}

// classifyCANopenMessage classifies a CAN ID into CANopen message type and extracts node ID
func classifyCANopenMessage(canID uint32) (string, uint32) {
	functionCode := canID >> 7
	nodeID := canID & 0x7F

	switch functionCode {
	case 0x0: // NMT
		return "nmt", nodeID
	case 0x1: // SYNC/EMCY
		if nodeID == 0 {
			return "sync", 0
		}
		return "emcy", nodeID
	case 0x3, 0x5, 0x7, 0x9: // TPDO1-4
		return "tpdo", nodeID
	case 0x4, 0x6, 0x8, 0xA: // RPDO1-4
		return "rpdo", nodeID
	case 0xB: // SDO (tx)
		return "sdo", nodeID
	case 0xC: // SDO (rx)
		return "sdo", nodeID
	case 0xE: // Heartbeat
		return "heartbeat", nodeID
	default:
		return "unknown", nodeID
	}
}
