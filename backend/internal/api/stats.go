package api

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// StatsAPI handles HTTP API requests for SocketCAN statistics
type StatsAPI struct {
	conn      driver.Conn
	tableName string
}

// NewStatsAPI creates a new Statistics API handler
func NewStatsAPI(conn driver.Conn, tableName string) *StatsAPI {
	return &StatsAPI{
		conn:      conn,
		tableName: tableName,
	}
}

// GetLatestStats retrieves the latest statistics for each interface
// GET /api/stats/latest?interface=can0
func (api *StatsAPI) GetLatestStats(w http.ResponseWriter, r *http.Request) {
	interfaceName := r.URL.Query().Get("interface")

	query := fmt.Sprintf(`
		SELECT
			timestamp, interface, state, mtu, queue_length,
			bitrate, sample_point, time_quanta, prop_seg, phase_seg1, phase_seg2,
			sjw, brp, restart_ms, controller_mode, bus_state,
			bus_error_counter, rx_error_counter, tx_error_counter,
			rx_packets, rx_bytes, rx_errors, rx_dropped, rx_over_errors,
			rx_crc_errors, rx_frame_errors, rx_fifo_errors, rx_missed,
			tx_packets, tx_bytes, tx_errors, tx_dropped, tx_aborted_errors,
			tx_carrier_errors, tx_fifo_errors, tx_heartbeat_errors, tx_window_errors,
			tx_aborted_restarts, tx_bus_error_restarts,
			collisions, carrier_changes, bus_off_restarts, arbitration_lost,
			error_warning, error_passive, bus_off
		FROM %s
		WHERE 1=1`, api.tableName)

	args := []any{}

	if interfaceName != "" {
		query += " AND interface = ?"
		args = append(args, interfaceName)
	}

	query += " ORDER BY timestamp DESC LIMIT 1"

	ctx := context.Background()
	row := api.conn.QueryRow(ctx, query, args...)

	var stat models.SocketCANStats
	err := row.Scan(
		&stat.Timestamp, &stat.Interface, &stat.State, &stat.MTU, &stat.QueueLength,
		&stat.Bitrate, &stat.SamplePoint, &stat.TimeQuanta, &stat.PropSeg, &stat.PhaseSeg1, &stat.PhaseSeg2,
		&stat.SJW, &stat.BRP, &stat.RestartMS, &stat.ControllerMode, &stat.BusState,
		&stat.BusErrorCounter, &stat.RXErrorCounter, &stat.TXErrorCounter,
		&stat.RXPackets, &stat.RXBytes, &stat.RXErrors, &stat.RXDropped, &stat.RXOverErrors,
		&stat.RXCRCErrors, &stat.RXFrameErrors, &stat.RXFIFOErrors, &stat.RXMissed,
		&stat.TXPackets, &stat.TXBytes, &stat.TXErrors, &stat.TXDropped, &stat.TXAbortedErrors,
		&stat.TXCarrierErrors, &stat.TXFIFOErrors, &stat.TXHeartbeatErrors, &stat.TXWindowErrors,
		&stat.TXAbortedRestarts, &stat.TXBusErrorRestarts,
		&stat.Collisions, &stat.CarrierChanges, &stat.BusOffRestarts, &stat.ArbitrationLost,
		&stat.ErrorWarning, &stat.ErrorPassive, &stat.BusOff,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, stat)
}

// GetStatsHistory retrieves historical statistics
// GET /api/stats/history?interface=can0&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&limit=100
func (api *StatsAPI) GetStatsHistory(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := fmt.Sprintf(`
		SELECT
			timestamp, interface, state, mtu, queue_length,
			bitrate, sample_point, time_quanta, prop_seg, phase_seg1, phase_seg2,
			sjw, brp, restart_ms, controller_mode, bus_state,
			bus_error_counter, rx_error_counter, tx_error_counter,
			rx_packets, rx_bytes, rx_errors, rx_dropped, rx_over_errors,
			rx_crc_errors, rx_frame_errors, rx_fifo_errors, rx_missed,
			tx_packets, tx_bytes, tx_errors, tx_dropped, tx_aborted_errors,
			tx_carrier_errors, tx_fifo_errors, tx_heartbeat_errors, tx_window_errors,
			tx_aborted_restarts, tx_bus_error_restarts,
			collisions, carrier_changes, bus_off_restarts, arbitration_lost,
			error_warning, error_passive, bus_off
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

	query += " ORDER BY timestamp DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	} else {
		query += " LIMIT 100" // Default limit
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

	stats := []models.SocketCANStats{}
	for rows.Next() {
		var stat models.SocketCANStats
		err := rows.Scan(
			&stat.Timestamp, &stat.Interface, &stat.State, &stat.MTU, &stat.QueueLength,
			&stat.Bitrate, &stat.SamplePoint, &stat.TimeQuanta, &stat.PropSeg, &stat.PhaseSeg1, &stat.PhaseSeg2,
			&stat.SJW, &stat.BRP, &stat.RestartMS, &stat.ControllerMode, &stat.BusState,
			&stat.BusErrorCounter, &stat.RXErrorCounter, &stat.TXErrorCounter,
			&stat.RXPackets, &stat.RXBytes, &stat.RXErrors, &stat.RXDropped, &stat.RXOverErrors,
			&stat.RXCRCErrors, &stat.RXFrameErrors, &stat.RXFIFOErrors, &stat.RXMissed,
			&stat.TXPackets, &stat.TXBytes, &stat.TXErrors, &stat.TXDropped, &stat.TXAbortedErrors,
			&stat.TXCarrierErrors, &stat.TXFIFOErrors, &stat.TXHeartbeatErrors, &stat.TXWindowErrors,
			&stat.TXAbortedRestarts, &stat.TXBusErrorRestarts,
			&stat.Collisions, &stat.CarrierChanges, &stat.BusOffRestarts, &stat.ArbitrationLost,
			&stat.ErrorWarning, &stat.ErrorPassive, &stat.BusOff,
		)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		stats = append(stats, stat)
	}

	respondWithJSON(w, http.StatusOK, stats)
}

// GetStatsAggregated retrieves aggregated statistics over a time period
// GET /api/stats/aggregated?interface=can0&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&interval=1h
func (api *StatsAPI) GetStatsAggregated(w http.ResponseWriter, r *http.Request) {
	params, err := parseQueryParams(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h" // Default to 1 hour
	}

	// Convert interval to ClickHouse interval format
	var clickhouseInterval string
	switch interval {
	case "1m", "1min":
		clickhouseInterval = "toStartOfMinute(timestamp)"
	case "5m", "5min":
		clickhouseInterval = "toStartOfFiveMinutes(timestamp)"
	case "15m", "15min":
		clickhouseInterval = "toStartOfFifteenMinutes(timestamp)"
	case "1h", "1hour":
		clickhouseInterval = "toStartOfHour(timestamp)"
	case "1d", "1day":
		clickhouseInterval = "toStartOfDay(timestamp)"
	default:
		clickhouseInterval = "toStartOfHour(timestamp)"
	}

	query := fmt.Sprintf(`
		SELECT
			%s as time_bucket,
			interface,
			avg(rx_packets) as avg_rx_packets,
			avg(tx_packets) as avg_tx_packets,
			avg(rx_bytes) as avg_rx_bytes,
			avg(tx_bytes) as avg_tx_bytes,
			sum(rx_errors) as total_rx_errors,
			sum(tx_errors) as total_tx_errors,
			sum(rx_dropped) as total_rx_dropped,
			sum(tx_dropped) as total_tx_dropped,
			max(bus_error_counter) as max_bus_error_counter,
			max(rx_error_counter) as max_rx_error_counter,
			max(tx_error_counter) as max_tx_error_counter
		FROM %s
		WHERE 1=1`, clickhouseInterval, api.tableName)

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

	query += " GROUP BY time_bucket, interface ORDER BY time_bucket DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	} else {
		query += " LIMIT 100"
	}

	ctx := context.Background()
	rows, err := api.conn.Query(ctx, query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}
	defer rows.Close()

	type AggregatedStats struct {
		TimeBucket          time.Time `json:"time_bucket"`
		Interface           string    `json:"interface"`
		AvgRXPackets        float64   `json:"avg_rx_packets"`
		AvgTXPackets        float64   `json:"avg_tx_packets"`
		AvgRXBytes          float64   `json:"avg_rx_bytes"`
		AvgTXBytes          float64   `json:"avg_tx_bytes"`
		TotalRXErrors       uint64    `json:"total_rx_errors"`
		TotalTXErrors       uint64    `json:"total_tx_errors"`
		TotalRXDropped      uint64    `json:"total_rx_dropped"`
		TotalTXDropped      uint64    `json:"total_tx_dropped"`
		MaxBusErrorCounter  int       `json:"max_bus_error_counter"`
		MaxRXErrorCounter   int       `json:"max_rx_error_counter"`
		MaxTXErrorCounter   int       `json:"max_tx_error_counter"`
	}

	aggregated := []AggregatedStats{}
	for rows.Next() {
		var agg AggregatedStats
		err := rows.Scan(
			&agg.TimeBucket, &agg.Interface,
			&agg.AvgRXPackets, &agg.AvgTXPackets,
			&agg.AvgRXBytes, &agg.AvgTXBytes,
			&agg.TotalRXErrors, &agg.TotalTXErrors,
			&agg.TotalRXDropped, &agg.TotalTXDropped,
			&agg.MaxBusErrorCounter, &agg.MaxRXErrorCounter, &agg.MaxTXErrorCounter,
		)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Scan failed: %v", err))
			return
		}

		aggregated = append(aggregated, agg)
	}

	respondWithJSON(w, http.StatusOK, aggregated)
}
