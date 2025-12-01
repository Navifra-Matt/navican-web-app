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

	query := fmt.Sprintf("SELECT timestamp, interface, can_id, dlc, data_0, data_1, data_2, data_3, data_4, data_5, data_6, data_7, data_hex FROM %s WHERE 1=1", api.tableName)
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
		var dlc uint8
		var data0, data1, data2, data3, data4, data5, data6, data7 uint8
		var dataHex string

		err := rows.Scan(&timestamp, &iface, &canID, &dlc, &data0, &data1, &data2, &data3, &data4, &data5, &data6, &data7, &dataHex)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		msg := models.CANMessageResponse{
			Timestamp: timestamp,
			Interface: iface,
			CANID:     canID,
			CANIDHex:  fmt.Sprintf("0x%X", canID),
			DLC:       dlc,
			Data:      []uint8{data0, data1, data2, data3, data4, data5, data6, data7},
			DataHex:   dataHex,
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
