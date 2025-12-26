package api

import (
	"can-db-writer/internal/database/clickhouse"
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseAPI handles HTTP API requests for ClickHouse data
type ClickHouseAPI struct {
	conn      driver.Conn
	tableName string
	writer    *clickhouse.Writer
}

// NewClickHouseAPI creates a new ClickHouse API handler
func NewClickHouseAPI(conn driver.Conn, tableName string, writer *clickhouse.Writer) *ClickHouseAPI {
	return &ClickHouseAPI{
		conn:      conn,
		tableName: tableName,
		writer:    writer,
	}
}

// GetCANopenMessages retrieves CAN messages classified by CANopen message type
// GET /api/clickhouse/canopen/messages?message_type=pdo&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&interface=can0&limit=100&offset=0
// message_type can be: nmt, sync, emcy, pdo, sdo, or empty for all
// Multiple message types: message_type=pdo&message_type=sdo&message_type=nmt
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

	// message_type can be single value or comma-separated array
	// e.g., message_type=pdo or message_type=pdo,sdo,nmt
	messageTypes := r.URL.Query()["message_type"]
	if len(messageTypes) == 0 {
		// Try comma-separated format
		if mt := r.URL.Query().Get("message_type"); mt != "" {
			messageTypes = []string{mt}
		}
	}

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
			CAST(CASE
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
			END AS UInt8) as node_id
		FROM %s
		WHERE 1=1`, api.tableName)
	args := []any{}

	// Add message type filter (supports multiple types)
	if len(messageTypes) > 0 {
		conditions := []string{}
		for _, messageType := range messageTypes {
			switch messageType {
			case "nmt":
				conditions = append(conditions, "can_id = 0x000")
			case "sync":
				conditions = append(conditions, "can_id = 0x080")
			case "emcy":
				conditions = append(conditions, "(can_id >= 0x081 AND can_id <= 0x0FF)")
			case "pdo":
				conditions = append(conditions, "((can_id >= 0x180 AND can_id <= 0x1FF) OR (can_id >= 0x200 AND can_id <= 0x27F) OR (can_id >= 0x280 AND can_id <= 0x2FF) OR (can_id >= 0x300 AND can_id <= 0x37F) OR (can_id >= 0x380 AND can_id <= 0x3FF) OR (can_id >= 0x400 AND can_id <= 0x47F) OR (can_id >= 0x480 AND can_id <= 0x4FF) OR (can_id >= 0x500 AND can_id <= 0x57F))")
			case "sdo":
				conditions = append(conditions, "((can_id >= 0x580 AND can_id <= 0x5FF) OR (can_id >= 0x600 AND can_id <= 0x67F))")
			case "heartbeat":
				conditions = append(conditions, "(can_id >= 0x700 AND can_id <= 0x77F)")
			}
		}
		if len(conditions) > 0 {
			query += " AND (" + conditions[0]
			for i := 1; i < len(conditions); i++ {
				query += " OR " + conditions[i]
			}
			query += ")"
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

// ExportData exports CAN messages to Parquet or Iceberg format
// POST /api/clickhouse/export
// Request body:
// {
//   "start_time": "2024-01-01T00:00:00Z",
//   "end_time": "2024-01-02T00:00:00Z",
//   "format": "parquet|iceberg" (optional, default: parquet),
//   "filename": "export.parquet" (optional, default: can_messages_YYYYMMDD.parquet or .iceberg),
//   "compression": "snappy|lz4|brotli|zstd|gzip|none" (optional, default: zstd)
// }
// Response: File download in the requested format
func (api *ClickHouseAPI) ExportData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
		Format      string `json:"format"`
		Filename    string `json:"filename"`
		Compression string `json:"compression"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid start_time format: %v", err))
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid end_time format: %v", err))
		return
	}

	// Set default format
	if req.Format == "" {
		req.Format = "parquet"
	}

	// Validate format
	var exportFormat clickhouse.ExportFormat
	var defaultExt string
	switch req.Format {
	case "parquet":
		exportFormat = clickhouse.FormatParquet
		defaultExt = ".parquet"
	case "iceberg":
		exportFormat = clickhouse.FormatIceberg
		defaultExt = ".iceberg"
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid format. Must be one of: parquet, iceberg")
		return
	}

	// Set default compression (as per ClickHouse documentation: zstd)
	if req.Compression == "" {
		req.Compression = "zstd"
	}

	// Validate compression (ClickHouse supported codecs)
	validCompressions := map[string]bool{
		"snappy": true,
		"lz4":    true,
		"brotli": true,
		"zstd":   true,
		"gzip":   true,
		"none":   true,
	}
	if !validCompressions[req.Compression] {
		respondWithError(w, http.StatusBadRequest, "Invalid compression. Must be one of: snappy, lz4, brotli, zstd, gzip, none")
		return
	}

	// Generate filename if not provided
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("can_messages_%s%s", startTime.Format("20060102"), defaultExt)
	}
	// Ensure correct extension
	if filepath.Ext(filename) != defaultExt {
		filename += defaultExt
	}

	// Create export options
	opts := clickhouse.ExportOptions{
		Format:      exportFormat,
		StartTime:   startTime,
		EndTime:     endTime,
		Compression: req.Compression,
	}

	// Set HTTP headers for file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Stream the export directly to the HTTP response
	if err := api.writer.ExportToWriter(w, api.tableName, opts); err != nil {
		// Note: If an error occurs after we start writing to w, we can't send a proper error response
		// The client will receive a partial file
		fmt.Printf("Export error: %v\n", err)
		return
	}
}
