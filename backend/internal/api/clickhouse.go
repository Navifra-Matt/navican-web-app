package api

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseAPI handles HTTP API requests for ClickHouse data
type ClickHouseAPI struct {
	conn      driver.Conn
	tableName string
}

// NewClickHouseAPI creates a new ClickHouse API handler
func NewClickHouseAPI(conn driver.Conn, tableName string) *ClickHouseAPI {
	return &ClickHouseAPI{
		conn:      conn,
		tableName: tableName,
	}
}

// GetMessages retrieves CAN messages with optional filters
// GET /api/clickhouse/messages?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&can_id=0x123&interface=can0&limit=100&offset=0
func (api *ClickHouseAPI) GetMessages(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf("SELECT timestamp, interface, can_id, data FROM %s WHERE 1=1", api.tableName)
	args := []any{}

	// Add filters
	if params.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *params.StartTime)
	}
	if params.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *params.EndTime)
	}
	if params.CANID != nil {
		query += " AND can_id = ?"
		args = append(args, *params.CANID)
	}
	if params.Interface != "" {
		query += " AND interface = ?"
		args = append(args, params.Interface)
	}

	query += " ORDER BY timestamp DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	}
	if params.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, params.Offset)
	}

	ctx := context.Background()
	rows, err := api.conn.Query(ctx, query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer rows.Close()

	messages := []models.CANMessageResponse{}
	for rows.Next() {
		var timestamp time.Time
		var iface string
		var canID uint32
		var data []uint8

		err := rows.Scan(&timestamp, &iface, &canID, &data)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		msg := models.CANMessageResponse{
			Timestamp: timestamp,
			Interface: iface,
			CANID:     canID,
			CANIDHex:  fmt.Sprintf("0x%X", canID),
			Data:      data,
		}
		messages = append(messages, msg)
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// GetMessageCount returns the count of messages with optional filters
// GET /api/clickhouse/count?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&can_id=0x123&interface=can0
func (api *ClickHouseAPI) GetMessageCount(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE 1=1", api.tableName)
	args := []any{}

	if params.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *params.StartTime)
	}
	if params.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *params.EndTime)
	}
	if params.CANID != nil {
		query += " AND can_id = ?"
		args = append(args, *params.CANID)
	}
	if params.Interface != "" {
		query += " AND interface = ?"
		args = append(args, params.Interface)
	}

	ctx := context.Background()
	var count uint64
	err = api.conn.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]uint64{"count": count})
}

// GetUniqueCANIDs returns all unique CAN IDs
// GET /api/clickhouse/can_ids?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&interface=can0
func (api *ClickHouseAPI) GetUniqueCANIDs(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf("SELECT DISTINCT can_id FROM %s WHERE 1=1", api.tableName)
	args := []any{}

	if params.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *params.StartTime)
	}
	if params.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *params.EndTime)
	}
	if params.Interface != "" {
		query += " AND interface = ?"
		args = append(args, params.Interface)
	}

	query += " ORDER BY can_id"

	ctx := context.Background()
	rows, err := api.conn.Query(ctx, query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer rows.Close()

	canIDs := []map[string]any{}
	for rows.Next() {
		var canID uint32
		err := rows.Scan(&canID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		canIDs = append(canIDs, map[string]any{
			"can_id":     canID,
			"can_id_hex": fmt.Sprintf("0x%X", canID),
		})
	}

	respondWithJSON(w, http.StatusOK, canIDs)
}

// GetStatsByCANID returns statistics grouped by CAN ID
// GET /api/clickhouse/stats?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&interface=can0
func (api *ClickHouseAPI) GetStatsByCANID(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf(`
		SELECT
			can_id,
			count(*) as message_count,
			min(timestamp) as first_seen,
			max(timestamp) as last_seen
		FROM %s
		WHERE 1=1`, api.tableName)
	args := []any{}

	if params.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *params.StartTime)
	}
	if params.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *params.EndTime)
	}
	if params.Interface != "" {
		query += " AND interface = ?"
		args = append(args, params.Interface)
	}

	query += " GROUP BY can_id ORDER BY message_count DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	}

	ctx := context.Background()
	rows, err := api.conn.Query(ctx, query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer rows.Close()

	stats := []map[string]any{}
	for rows.Next() {
		var canID uint32
		var count uint64
		var firstSeen, lastSeen time.Time

		err := rows.Scan(&canID, &count, &firstSeen, &lastSeen)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		stats = append(stats, map[string]any{
			"can_id":        canID,
			"can_id_hex":    fmt.Sprintf("0x%X", canID),
			"message_count": count,
			"first_seen":    firstSeen,
			"last_seen":     lastSeen,
		})
	}

	respondWithJSON(w, http.StatusOK, stats)
}

// GetCANopenMessages retrieves CAN messages classified by CANopen message type
// GET /api/clickhouse/canopen/messages?message_type=pdo&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&interface=can0&limit=100&offset=0
// message_type can be: nmt, sync, emcy, pdo, sdo, or empty for all
//
// Dynamic PDO field mapping via query parameters:
// tpdo1=statusword:uint16:0:2,mode_of_operation:int8:2:1
// tpdo2=actual_velocity:int32:0:4,actual_position:int32:4:4
// rpdo1=control_word:uint16:0:2,target_position:int32:2:4
// Format: field_name:type:byte_offset:byte_length
// Types: int8, uint8, int16, uint16, int32, uint32
//
// node_id filter: node_id=1 (filter by specific CANopen node ID)
func (api *ClickHouseAPI) GetCANopenMessages(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	messageType := r.URL.Query().Get("message_type")
	nodeIDStr := r.URL.Query().Get("node_id")

	var nodeIDFilter *uint8
	if nodeIDStr != "" {
		var nodeID uint64
		_, err := fmt.Sscanf(nodeIDStr, "%d", &nodeID)
		if err == nil && nodeID <= 127 {
			n := uint8(nodeID)
			nodeIDFilter = &n
		}
	}

	// Parse dynamic PDO mappings from query parameters
	queryMappings := make(map[string]*models.PDOMapping)

	// TPDO (Transmit PDO) - TX direction
	for pdoNum := 1; pdoNum <= 4; pdoNum++ {
		paramName := fmt.Sprintf("tpdo%d", pdoNum)
		if fieldsStr := r.URL.Query().Get(paramName); fieldsStr != "" {
			fields, err := models.ParsePDOFieldsFromQuery(fieldsStr)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid %s: %v", paramName, err))
				return
			}
			key := fmt.Sprintf("tpdo%d", pdoNum)
			queryMappings[key] = models.CreatePDOMappingFromQuery(pdoNum, "TX", fields)
		}
	}

	// RPDO (Receive PDO) - RX direction
	for pdoNum := 1; pdoNum <= 4; pdoNum++ {
		paramName := fmt.Sprintf("rpdo%d", pdoNum)
		if fieldsStr := r.URL.Query().Get(paramName); fieldsStr != "" {
			fields, err := models.ParsePDOFieldsFromQuery(fieldsStr)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid %s: %v", paramName, err))
				return
			}
			key := fmt.Sprintf("rpdo%d", pdoNum)
			queryMappings[key] = models.CreatePDOMappingFromQuery(pdoNum, "RX", fields)
		}
	}

	// Build query with CANopen message type classification
	query := fmt.Sprintf(`
		SELECT
			timestamp,
			interface,
			can_id,
			data,
			CASE
				WHEN can_id = 0x000 THEN 'NMT'
				WHEN can_id = 0x080 THEN 'SYNC'
				WHEN can_id >= 0x081 AND can_id <= 0x0FF THEN 'EMCY'
				WHEN can_id >= 0x180 AND can_id <= 0x1FF THEN 'TPDO1'
				WHEN can_id >= 0x200 AND can_id <= 0x27F THEN 'RPDO1'
				WHEN can_id >= 0x280 AND can_id <= 0x2FF THEN 'TPDO2'
				WHEN can_id >= 0x300 AND can_id <= 0x37F THEN 'RPDO2'
				WHEN can_id >= 0x380 AND can_id <= 0x3FF THEN 'TPDO3'
				WHEN can_id >= 0x400 AND can_id <= 0x47F THEN 'RPDO3'
				WHEN can_id >= 0x480 AND can_id <= 0x4FF THEN 'TPDO4'
				WHEN can_id >= 0x500 AND can_id <= 0x57F THEN 'RPDO4'
				WHEN can_id >= 0x580 AND can_id <= 0x5FF THEN 'SDO_TX'
				WHEN can_id >= 0x600 AND can_id <= 0x67F THEN 'SDO_RX'
				WHEN can_id >= 0x700 AND can_id <= 0x77F THEN 'HEARTBEAT'
				ELSE 'UNKNOWN'
			END as message_type,
			CASE
				WHEN can_id = 0x000 OR can_id = 0x080 THEN 0
				WHEN can_id >= 0x081 AND can_id <= 0x0FF THEN can_id - 0x080
				WHEN can_id >= 0x180 AND can_id <= 0x1FF THEN can_id - 0x180 + 1
				WHEN can_id >= 0x200 AND can_id <= 0x27F THEN can_id - 0x200 + 1
				WHEN can_id >= 0x280 AND can_id <= 0x2FF THEN can_id - 0x280 + 1
				WHEN can_id >= 0x300 AND can_id <= 0x37F THEN can_id - 0x300 + 1
				WHEN can_id >= 0x380 AND can_id <= 0x3FF THEN can_id - 0x380 + 1
				WHEN can_id >= 0x400 AND can_id <= 0x47F THEN can_id - 0x400 + 1
				WHEN can_id >= 0x480 AND can_id <= 0x4FF THEN can_id - 0x480 + 1
				WHEN can_id >= 0x500 AND can_id <= 0x57F THEN can_id - 0x500 + 1
				WHEN can_id >= 0x580 AND can_id <= 0x5FF THEN can_id - 0x580 + 1
				WHEN can_id >= 0x600 AND can_id <= 0x67F THEN can_id - 0x600 + 1
				WHEN can_id >= 0x700 AND can_id <= 0x77F THEN can_id - 0x700 + 1
				ELSE 0
			END as node_id
		FROM %s
		WHERE 1=1`, api.tableName)
	args := []any{}

	// Add message type filter
	if messageType != "" {
		switch messageType {
		case "nmt":
			query += " AND can_id = 0x000"
		case "sync":
			query += " AND can_id = 0x080"
		case "emcy":
			query += " AND can_id >= 0x081 AND can_id <= 0x0FF"
		case "pdo":
			query += " AND ((can_id >= 0x180 AND can_id <= 0x1FF) OR (can_id >= 0x200 AND can_id <= 0x27F) OR (can_id >= 0x280 AND can_id <= 0x2FF) OR (can_id >= 0x300 AND can_id <= 0x37F) OR (can_id >= 0x380 AND can_id <= 0x3FF) OR (can_id >= 0x400 AND can_id <= 0x47F) OR (can_id >= 0x480 AND can_id <= 0x4FF) OR (can_id >= 0x500 AND can_id <= 0x57F))"
		case "sdo":
			query += " AND ((can_id >= 0x580 AND can_id <= 0x5FF) OR (can_id >= 0x600 AND can_id <= 0x67F))"
		case "heartbeat":
			query += " AND can_id >= 0x700 AND can_id <= 0x77F"
		}
	}

	// Add other filters
	if params.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *params.StartTime)
	}
	if params.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *params.EndTime)
	}
	if params.CANID != nil {
		query += " AND can_id = ?"
		args = append(args, *params.CANID)
	}
	if params.Interface != "" {
		query += " AND interface = ?"
		args = append(args, params.Interface)
	}

	// Add node_id filter if specified
	if nodeIDFilter != nil {
		query += " AND ("
		query += "    (can_id >= 0x081 AND can_id <= 0x0FF AND can_id - 0x080 = ?) OR"
		query += "    (can_id >= 0x180 AND can_id <= 0x1FF AND can_id - 0x180 + 1 = ?) OR"
		query += "    (can_id >= 0x200 AND can_id <= 0x27F AND can_id - 0x200 + 1 = ?) OR"
		query += "    (can_id >= 0x280 AND can_id <= 0x2FF AND can_id - 0x280 + 1 = ?) OR"
		query += "    (can_id >= 0x300 AND can_id <= 0x37F AND can_id - 0x300 + 1 = ?) OR"
		query += "    (can_id >= 0x380 AND can_id <= 0x3FF AND can_id - 0x380 + 1 = ?) OR"
		query += "    (can_id >= 0x400 AND can_id <= 0x47F AND can_id - 0x400 + 1 = ?) OR"
		query += "    (can_id >= 0x480 AND can_id <= 0x4FF AND can_id - 0x480 + 1 = ?) OR"
		query += "    (can_id >= 0x500 AND can_id <= 0x57F AND can_id - 0x500 + 1 = ?) OR"
		query += "    (can_id >= 0x580 AND can_id <= 0x5FF AND can_id - 0x580 + 1 = ?) OR"
		query += "    (can_id >= 0x600 AND can_id <= 0x67F AND can_id - 0x600 + 1 = ?) OR"
		query += "    (can_id >= 0x700 AND can_id <= 0x77F AND can_id - 0x700 + 1 = ?)"
		query += ")"
		// Add the node_id 12 times for each condition
		for i := 0; i < 12; i++ {
			args = append(args, *nodeIDFilter)
		}
	}

	query += " ORDER BY timestamp DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	}
	if params.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, params.Offset)
	}

	ctx := context.Background()
	rows, err := api.conn.Query(ctx, query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer rows.Close()

	messages := []map[string]any{}
	for rows.Next() {
		var timestamp time.Time
		var iface string
		var canID uint32
		var dataBytes []uint8
		var msgType string
		var nodeID uint8

		err := rows.Scan(&timestamp, &iface, &canID, &dataBytes, &msgType, &nodeID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		msg := map[string]any{
			"timestamp":    timestamp,
			"interface":    iface,
			"can_id":       canID,
			"can_id_hex":   fmt.Sprintf("0x%X", canID),
			"data":         dataBytes,
			"message_type": msgType,
			"node_id":      nodeID,
		}

		// Parse PDO data if this is a PDO message and query mapping is provided
		pdoType := models.GetPDOMessageType(canID)
		if pdoType != nil {
			// Convert to lowercase tpdo/rpdo format to match query parameter names
			var key string
			if pdoType.Direction == "TX" {
				key = fmt.Sprintf("tpdo%d", pdoType.PDONumber)
			} else {
				key = fmt.Sprintf("rpdo%d", pdoType.PDONumber)
			}
			mapping := queryMappings[key]

			if mapping != nil {
				parsedData := mapping.ParsePDOData(dataBytes)
				msg["parsed_data"] = parsedData
			}
		}

		messages = append(messages, msg)
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// GetCANopenStats returns statistics grouped by CANopen message type
// GET /api/clickhouse/canopen/stats?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&interface=can0
func (api *ClickHouseAPI) GetCANopenStats(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf(`
		SELECT
			CASE
				WHEN can_id = 0x000 THEN 'NMT'
				WHEN can_id = 0x080 THEN 'SYNC'
				WHEN can_id >= 0x081 AND can_id <= 0x0FF THEN 'EMCY'
				WHEN can_id >= 0x180 AND can_id <= 0x1FF THEN 'TPDO1'
				WHEN can_id >= 0x200 AND can_id <= 0x27F THEN 'RPDO1'
				WHEN can_id >= 0x280 AND can_id <= 0x2FF THEN 'TPDO2'
				WHEN can_id >= 0x300 AND can_id <= 0x37F THEN 'RPDO2'
				WHEN can_id >= 0x380 AND can_id <= 0x3FF THEN 'TPDO3'
				WHEN can_id >= 0x400 AND can_id <= 0x47F THEN 'RPDO3'
				WHEN can_id >= 0x480 AND can_id <= 0x4FF THEN 'TPDO4'
				WHEN can_id >= 0x500 AND can_id <= 0x57F THEN 'RPDO4'
				WHEN can_id >= 0x580 AND can_id <= 0x5FF THEN 'SDO_TX'
				WHEN can_id >= 0x600 AND can_id <= 0x67F THEN 'SDO_RX'
				WHEN can_id >= 0x700 AND can_id <= 0x77F THEN 'HEARTBEAT'
				ELSE 'UNKNOWN'
			END as message_type,
			count(*) as message_count,
			min(timestamp) as first_seen,
			max(timestamp) as last_seen
		FROM %s
		WHERE 1=1`, api.tableName)
	args := []any{}

	if params.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *params.StartTime)
	}
	if params.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *params.EndTime)
	}
	if params.Interface != "" {
		query += " AND interface = ?"
		args = append(args, params.Interface)
	}

	query += " GROUP BY message_type ORDER BY message_count DESC"

	ctx := context.Background()
	rows, err := api.conn.Query(ctx, query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer rows.Close()

	stats := []map[string]any{}
	for rows.Next() {
		var msgType string
		var count uint64
		var firstSeen, lastSeen time.Time

		err := rows.Scan(&msgType, &count, &firstSeen, &lastSeen)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		stats = append(stats, map[string]any{
			"message_type":  msgType,
			"message_count": count,
			"first_seen":    firstSeen,
			"last_seen":     lastSeen,
		})
	}

	respondWithJSON(w, http.StatusOK, stats)
}
