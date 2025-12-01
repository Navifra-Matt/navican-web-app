package api

import (
	"can-db-writer/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// InfluxDBAPI handles HTTP API requests for InfluxDB data
type InfluxDBAPI struct {
	client influxdb2.Client
	org    string
	bucket string
}

// NewInfluxDBAPI creates a new InfluxDB API handler
func NewInfluxDBAPI(url, token, org, bucket string) (*InfluxDBAPI, error) {
	client := influxdb2.NewClient(url, token)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return nil, fmt.Errorf("InfluxDB health check failed: %s", msg)
	}

	return &InfluxDBAPI{
		client: client,
		org:    org,
		bucket: bucket,
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

	// Build Flux query
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r["_measurement"] == "can_messages")
	`, api.bucket, getStartTime(params), getStopTime(params))

	// Add filters
	if params.Interface != "" {
		query += fmt.Sprintf(`|> filter(fn: (r) => r["interface"] == "%s")`, params.Interface)
	}

	if params.CANID != nil {
		query += fmt.Sprintf(`|> filter(fn: (r) => r["can_id"] == "0x%X")`, *params.CANID)
	}

	// Sort and limit
	query += `|> sort(columns: ["_time"], desc: true)`

	limit := 100
	if params.Limit > 0 {
		limit = params.Limit
	}
	query += fmt.Sprintf(`|> limit(n: %d)`, limit)

	// Execute query
	queryAPI := api.client.QueryAPI(api.org)
	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer result.Close()

	// Parse results
	messages := []models.CANMessageResponse{}
	currentMessage := make(map[string]any)
	var currentTime time.Time

	for result.Next() {
		record := result.Record()

		// If we have a new timestamp, save the previous message
		if !currentTime.IsZero() && !record.Time().Equal(currentTime) {
			if msg := buildMessageFromRecord(currentMessage, currentTime); msg != nil {
				messages = append(messages, *msg)
			}
			currentMessage = make(map[string]any)
		}

		currentTime = record.Time()
		field := record.Field()
		value := record.Value()

		currentMessage[field] = value

		// Store tags
		if iface, ok := record.ValueByKey("interface").(string); ok {
			currentMessage["interface"] = iface
		}
		if canID, ok := record.ValueByKey("can_id").(string); ok {
			currentMessage["can_id"] = canID
		}
	}

	// Don't forget the last message
	if !currentTime.IsZero() {
		if msg := buildMessageFromRecord(currentMessage, currentTime); msg != nil {
			messages = append(messages, *msg)
		}
	}

	if err := result.Err(); err != nil {
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

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r["_measurement"] == "can_messages")
		|> filter(fn: (r) => r["_field"] == "can_id_decimal")
	`, api.bucket, getStartTime(params), getStopTime(params))

	if params.Interface != "" {
		query += fmt.Sprintf(`|> filter(fn: (r) => r["interface"] == "%s")`, params.Interface)
	}

	if params.CANID != nil {
		query += fmt.Sprintf(`|> filter(fn: (r) => r["can_id"] == "0x%X")`, *params.CANID)
	}

	query += `|> count()`

	queryAPI := api.client.QueryAPI(api.org)
	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer result.Close()

	count := uint64(0)
	if result.Next() {
		if val, ok := result.Record().Value().(int64); ok {
			count = uint64(val)
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]uint64{"count": count})
}

// HealthCheck returns InfluxDB health status
// GET /api/influxdb/health
func (api *InfluxDBAPI) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := api.client.Health(ctx)
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("Health check failed: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{
		"status":  health.Status,
		"message": health.Message,
		"version": health.Version,
	})
}

// ExecuteQuery executes a custom Flux query
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

	queryAPI := api.client.QueryAPI(api.org)
	result, err := queryAPI.Query(context.Background(), req.Query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer result.Close()

	// Collect results
	results := []map[string]any{}
	for result.Next() {
		record := result.Record()
		row := make(map[string]any)

		row["time"] = record.Time()
		row["measurement"] = record.Measurement()
		row["field"] = record.Field()
		row["value"] = record.Value()

		// Add tags
		for k, v := range record.Values() {
			if k != "_time" && k != "_measurement" && k != "_field" && k != "_value" {
				row[k] = v
			}
		}

		results = append(results, row)
	}

	if err := result.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query error: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, results)
}

// Helper functions

func getStartTime(params models.QueryParams) string {
	if params.StartTime != nil {
		return params.StartTime.Format(time.RFC3339)
	}
	return "-1h" // Default to last 1 hour
}

func getStopTime(params models.QueryParams) string {
	if params.EndTime != nil {
		return params.EndTime.Format(time.RFC3339)
	}
	return "now()"
}

func buildMessageFromRecord(record map[string]any, timestamp time.Time) *models.CANMessageResponse {
	msg := &models.CANMessageResponse{
		Timestamp: timestamp,
		Data:      make([]uint8, 8),
	}

	// Extract interface
	if iface, ok := record["interface"].(string); ok {
		msg.Interface = iface
	}

	// Extract CAN ID
	if canIDHex, ok := record["can_id"].(string); ok {
		msg.CANIDHex = canIDHex
		var canID uint32
		fmt.Sscanf(canIDHex, "0x%X", &canID)
		msg.CANID = canID
	} else if canIDDecimal, ok := record["can_id_decimal"].(int64); ok {
		msg.CANID = uint32(canIDDecimal)
		msg.CANIDHex = fmt.Sprintf("0x%X", canIDDecimal)
	}

	// Extract DLC
	if dlc, ok := record["dlc"].(int64); ok {
		msg.DLC = uint8(dlc)
	}

	// Extract data bytes
	for i := 0; i < 8; i++ {
		field := fmt.Sprintf("data_%d", i)
		if val, ok := record[field].(int64); ok {
			msg.Data[i] = uint8(val)
		}
	}

	// Extract data_hex
	if dataHex, ok := record["data_hex"].(string); ok {
		msg.DataHex = dataHex
	} else {
		msg.DataHex = fmt.Sprintf("%02X %02X %02X %02X %02X %02X %02X %02X",
			msg.Data[0], msg.Data[1], msg.Data[2], msg.Data[3],
			msg.Data[4], msg.Data[5], msg.Data[6], msg.Data[7])
	}

	return msg
}

// Close closes the InfluxDB client
func (api *InfluxDBAPI) Close() {
	if api.client != nil {
		api.client.Close()
	}
}
