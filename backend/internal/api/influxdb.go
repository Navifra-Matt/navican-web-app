package api

import (
	"can-db-writer/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// InfluxDBAPI handles HTTP API requests for InfluxDB data
type InfluxDBAPI struct {
	client   *influxdb3.Client
	database string
}

// NewInfluxDBAPI creates a new InfluxDB API handler
func NewInfluxDBAPI(url, token, database string) (*InfluxDBAPI, error) {
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     url,
		Token:    token,
		Database: database,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create InfluxDB client: %w", err)
	}

	return &InfluxDBAPI{
		client:   client,
		database: database,
	}, nil
}

// GetMessages retrieves CAN messages with optional filters
// GET /api/influxdb/messages?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&can_id=0x123&interface=can0&limit=100
func (api *InfluxDBAPI) GetMessages(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build SQL query for InfluxDB v3
	query := fmt.Sprintf(`
		SELECT time, interface, can_id, can_id_decimal,
		       data_0, data_1, data_2, data_3, data_4, data_5, data_6, data_7
		FROM can_messages
		WHERE time >= '%s' AND time <= '%s'
	`, getSQLStartTime(params), getSQLStopTime(params))

	// Add filters
	if params.Interface != "" {
		query += fmt.Sprintf(` AND interface = '%s'`, params.Interface)
	}

	if params.CANID != nil {
		query += fmt.Sprintf(` AND can_id = '0x%X'`, *params.CANID)
	}

	// Sort and limit
	query += ` ORDER BY time DESC`

	limit := 100
	if params.Limit > 0 {
		limit = params.Limit
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	// Execute query using SQL
	iterator, err := api.client.Query(context.Background(), query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	// Parse results
	messages := []models.CANMessageResponse{}

	for iterator.Next() {
		record := iterator.Value()

		msg := &models.CANMessageResponse{
			Data: make([]uint8, 8),
		}

		// Extract fields from record
		if t, ok := record["time"].(time.Time); ok {
			msg.Timestamp = t
		}
		if iface, ok := record["interface"].(string); ok {
			msg.Interface = iface
		}
		if canIDHex, ok := record["can_id"].(string); ok {
			msg.CANIDHex = canIDHex
			var canID uint32
			fmt.Sscanf(canIDHex, "0x%X", &canID)
			msg.CANID = canID
		}
		if canIDDecimal, ok := record["can_id_decimal"].(int64); ok {
			msg.CANID = uint32(canIDDecimal)
			if msg.CANIDHex == "" {
				msg.CANIDHex = fmt.Sprintf("0x%X", canIDDecimal)
			}
		}

		// Extract data bytes
		for i := 0; i < 8; i++ {
			field := fmt.Sprintf("data_%d", i)
			if val, ok := record[field].(int64); ok {
				msg.Data[i] = uint8(val)
			} else if val, ok := record[field].(uint8); ok {
				msg.Data[i] = val
			}
		}

		msg.DataHex = fmt.Sprintf("%02X %02X %02X %02X %02X %02X %02X %02X",
			msg.Data[0], msg.Data[1], msg.Data[2], msg.Data[3],
			msg.Data[4], msg.Data[5], msg.Data[6], msg.Data[7])

		messages = append(messages, *msg)
	}

	if err := iterator.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query error: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// GetMessageCount returns the count of messages
// GET /api/influxdb/count?start_time=2024-01-01T00:00:00Z&can_id=0x123
func (api *InfluxDBAPI) GetMessageCount(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build SQL count query for InfluxDB v3
	query := fmt.Sprintf(`
		SELECT COUNT(*) as count
		FROM can_messages
		WHERE time >= '%s' AND time <= '%s'
	`, getSQLStartTime(params), getSQLStopTime(params))

	if params.Interface != "" {
		query += fmt.Sprintf(` AND interface = '%s'`, params.Interface)
	}

	if params.CANID != nil {
		query += fmt.Sprintf(` AND can_id = '0x%X'`, *params.CANID)
	}

	iterator, err := api.client.Query(context.Background(), query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	count := uint64(0)
	if iterator.Next() {
		record := iterator.Value()
		if val, ok := record["count"].(int64); ok {
			count = uint64(val)
		}
	}

	if err := iterator.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query error: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]uint64{"count": count})
}

// HealthCheck returns InfluxDB health status
// GET /api/influxdb/health
func (api *InfluxDBAPI) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection by running a simple query
	query := "SELECT 1"
	_, err := api.client.Query(ctx, query)
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("Health check failed: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{
		"status":  "healthy",
		"message": "InfluxDB v3 connection successful",
	})
}

// ExecuteQuery executes a custom SQL query
// POST /api/influxdb/query
func (api *InfluxDBAPI) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Query == "" {
		respondWithError(w, http.StatusBadRequest, "Query is required")
		return
	}

	iterator, err := api.client.Query(context.Background(), req.Query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	// Collect results
	results := []map[string]any{}
	for iterator.Next() {
		record := iterator.Value()
		results = append(results, record)
	}

	if err := iterator.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query error: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, results)
}

// Helper functions

func getSQLStartTime(params models.QueryParams) string {
	if params.StartTime != nil {
		return params.StartTime.Format(time.RFC3339Nano)
	}
	// Default to last 1 hour
	return time.Now().Add(-1 * time.Hour).Format(time.RFC3339Nano)
}

func getSQLStopTime(params models.QueryParams) string {
	if params.EndTime != nil {
		return params.EndTime.Format(time.RFC3339Nano)
	}
	return time.Now().Format(time.RFC3339Nano)
}

// Close closes the InfluxDB client
func (api *InfluxDBAPI) Close() {
	if api.client != nil {
		api.client.Close()
	}
}
