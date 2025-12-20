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

// GetMessages retrieves CAN messages with optional filters
func (s *CANServer) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
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
		if req.Filter.CanId != nil {
			query += " AND can_id = ?"
			args = append(args, *req.Filter.CanId)
		}
		if req.Filter.Interface != "" {
			query += " AND interface = ?"
			args = append(args, req.Filter.Interface)
		}
	}

	query += " ORDER BY timestamp DESC"

	if req.Filter != nil {
		if req.Filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, req.Filter.Limit)
		}
		if req.Filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, req.Filter.Offset)
		}
	}

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*pb.CANMessage
	for rows.Next() {
		var ts time.Time
		var iface string
		var canID uint32
		var canIDHex string
		var data []byte

		if err := rows.Scan(&ts, &iface, &canID, &canIDHex, &data); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		messages = append(messages, &pb.CANMessage{
			Timestamp: timestamppb.New(ts),
			Interface: iface,
			CanId:     canID,
			CanIdHex:  canIDHex,
			Data:      data,
		})
	}

	return &pb.GetMessagesResponse{Messages: messages}, nil
}

// GetMessageCount retrieves the count of messages matching the filters
func (s *CANServer) GetMessageCount(ctx context.Context, req *pb.GetMessageCountRequest) (*pb.GetMessageCountResponse, error) {
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE 1=1", s.tableName)
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
		if req.Filter.CanId != nil {
			query += " AND can_id = ?"
			args = append(args, *req.Filter.CanId)
		}
		if req.Filter.Interface != "" {
			query += " AND interface = ?"
			args = append(args, req.Filter.Interface)
		}
	}

	var count uint64
	if err := s.conn.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to query count: %w", err)
	}

	return &pb.GetMessageCountResponse{Count: count}, nil
}

// GetUniqueCANIDs retrieves unique CAN IDs
func (s *CANServer) GetUniqueCANIDs(ctx context.Context, req *pb.GetUniqueCANIDsRequest) (*pb.GetUniqueCANIDsResponse, error) {
	query := fmt.Sprintf("SELECT DISTINCT can_id, hex(can_id) as can_id_hex FROM %s WHERE 1=1", s.tableName)
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

	query += " ORDER BY can_id"

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique CAN IDs: %w", err)
	}
	defer rows.Close()

	var canIDs []*pb.CANIDInfo
	for rows.Next() {
		var canID uint32
		var canIDHex string
		if err := rows.Scan(&canID, &canIDHex); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		canIDs = append(canIDs, &pb.CANIDInfo{
			CanId:    canID,
			CanIdHex: canIDHex,
		})
	}

	return &pb.GetUniqueCANIDsResponse{CanIds: canIDs}, nil
}

// GetStatsByCANID retrieves statistics grouped by CAN ID
func (s *CANServer) GetStatsByCANID(ctx context.Context, req *pb.GetStatsByCANIDRequest) (*pb.GetStatsByCANIDResponse, error) {
	query := fmt.Sprintf(`
		SELECT
			can_id,
			hex(can_id) as can_id_hex,
			count(*) as message_count,
			min(timestamp) as first_seen,
			max(timestamp) as last_seen
		FROM %s
		WHERE 1=1`, s.tableName)
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

	query += " GROUP BY can_id ORDER BY message_count DESC"

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	var stats []*pb.CANIDStats
	for rows.Next() {
		var stat pb.CANIDStats
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&stat.CanId, &stat.CanIdHex, &stat.MessageCount, &firstSeen, &lastSeen); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		stat.FirstSeen = timestamppb.New(firstSeen)
		stat.LastSeen = timestamppb.New(lastSeen)
		stats = append(stats, &stat)
	}

	return &pb.GetStatsByCANIDResponse{Stats: stats}, nil
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

// GetCANopenStats retrieves statistics grouped by CANopen message type
func (s *CANServer) GetCANopenStats(ctx context.Context, req *pb.GetCANopenStatsRequest) (*pb.GetCANopenStatsResponse, error) {
	query := fmt.Sprintf("SELECT can_id, count(*) as message_count, min(timestamp) as first_seen, max(timestamp) as last_seen FROM %s WHERE 1=1", s.tableName)
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

	query += " GROUP BY can_id"

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	// Group by message type
	statsMap := make(map[string]*pb.CANopenMessageTypeStats)

	for rows.Next() {
		var canID uint32
		var count uint64
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&canID, &count, &firstSeen, &lastSeen); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		msgType, _ := classifyCANopenMessage(canID)

		if stat, exists := statsMap[msgType]; exists {
			stat.MessageCount += count
			if firstSeen.Before(stat.FirstSeen.AsTime()) {
				stat.FirstSeen = timestamppb.New(firstSeen)
			}
			if lastSeen.After(stat.LastSeen.AsTime()) {
				stat.LastSeen = timestamppb.New(lastSeen)
			}
		} else {
			statsMap[msgType] = &pb.CANopenMessageTypeStats{
				MessageType:  msgType,
				MessageCount: count,
				FirstSeen:    timestamppb.New(firstSeen),
				LastSeen:     timestamppb.New(lastSeen),
			}
		}
	}

	var stats []*pb.CANopenMessageTypeStats
	for _, stat := range statsMap {
		stats = append(stats, stat)
	}

	return &pb.GetCANopenStatsResponse{Stats: stats}, nil
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
