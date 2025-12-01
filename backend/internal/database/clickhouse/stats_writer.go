package clickhouse

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// StatsWriter handles writing SocketCAN statistics to ClickHouse
type StatsWriter struct {
	conn       driver.Conn
	batchSize  int
	batch      []models.SocketCANStats
	batchChan  chan models.SocketCANStats
	ctx        context.Context
	cancel     context.CancelFunc
	flushTimer *time.Ticker
}

// NewStatsWriter creates a new ClickHouse statistics writer
func NewStatsWriter(conn driver.Conn, batchSize int) *StatsWriter {
	ctx, cancel := context.WithCancel(context.Background())

	writer := &StatsWriter{
		conn:       conn,
		batchSize:  batchSize,
		batch:      make([]models.SocketCANStats, 0, batchSize),
		batchChan:  make(chan models.SocketCANStats, batchSize*2),
		ctx:        ctx,
		cancel:     cancel,
		flushTimer: time.NewTicker(5 * time.Second), // Flush every 5 seconds
	}

	return writer
}

// CreateStatsTable creates the SocketCAN statistics table in ClickHouse
func CreateStatsTable(conn driver.Conn, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			timestamp DateTime64(6),
			interface String,
			state String,
			mtu UInt32,
			queue_length UInt32,

			-- CAN-specific parameters
			bitrate UInt32,
			sample_point String,
			time_quanta UInt32,
			prop_seg UInt16,
			phase_seg1 UInt16,
			phase_seg2 UInt16,
			sjw UInt16,
			brp UInt16,
			restart_ms UInt32,
			controller_mode String,
			bus_state String,
			bus_error_counter UInt32,
			rx_error_counter UInt32,
			tx_error_counter UInt32,

			-- RX statistics
			rx_packets UInt64,
			rx_bytes UInt64,
			rx_errors UInt64,
			rx_dropped UInt64,
			rx_over_errors UInt64,
			rx_crc_errors UInt64,
			rx_frame_errors UInt64,
			rx_fifo_errors UInt64,
			rx_missed UInt64,

			-- TX statistics
			tx_packets UInt64,
			tx_bytes UInt64,
			tx_errors UInt64,
			tx_dropped UInt64,
			tx_aborted_errors UInt64,
			tx_carrier_errors UInt64,
			tx_fifo_errors UInt64,
			tx_heartbeat_errors UInt64,
			tx_window_errors UInt64,
			tx_aborted_restarts UInt64,
			tx_bus_error_restarts UInt64,

			-- Additional statistics
			collisions UInt64,
			carrier_changes UInt64,
			bus_off_restarts UInt64,
			arbitration_lost UInt64,
			error_warning UInt64,
			error_passive UInt64,
			bus_off UInt64
		) ENGINE = MergeTree()
		ORDER BY (timestamp, interface)
		PARTITION BY toYYYYMMDD(timestamp)
		SETTINGS index_granularity = 8192
	`, tableName)

	return conn.Exec(context.Background(), query)
}

// Start begins processing and writing statistics
func (w *StatsWriter) Start(tableName string) {
	go w.writeLoop(tableName)
}

// writeLoop processes statistics and writes them in batches
func (w *StatsWriter) writeLoop(tableName string) {
	for {
		select {
		case <-w.ctx.Done():
			// Flush remaining statistics before exiting
			if len(w.batch) > 0 {
				w.flush(tableName)
			}
			return

		case stat := <-w.batchChan:
			w.batch = append(w.batch, stat)
			if len(w.batch) >= w.batchSize {
				w.flush(tableName)
			}

		case <-w.flushTimer.C:
			if len(w.batch) > 0 {
				w.flush(tableName)
			}
		}
	}
}

// flush writes the current batch to ClickHouse
func (w *StatsWriter) flush(tableName string) error {
	if len(w.batch) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(w.ctx, fmt.Sprintf("INSERT INTO %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, stat := range w.batch {
		err = batch.Append(
			stat.Timestamp,
			stat.Interface,
			stat.State,
			stat.MTU,
			stat.QueueLength,
			stat.Bitrate,
			stat.SamplePoint,
			stat.TimeQuanta,
			stat.PropSeg,
			stat.PhaseSeg1,
			stat.PhaseSeg2,
			stat.SJW,
			stat.BRP,
			stat.RestartMS,
			stat.ControllerMode,
			stat.BusState,
			stat.BusErrorCounter,
			stat.RXErrorCounter,
			stat.TXErrorCounter,
			stat.RXPackets,
			stat.RXBytes,
			stat.RXErrors,
			stat.RXDropped,
			stat.RXOverErrors,
			stat.RXCRCErrors,
			stat.RXFrameErrors,
			stat.RXFIFOErrors,
			stat.RXMissed,
			stat.TXPackets,
			stat.TXBytes,
			stat.TXErrors,
			stat.TXDropped,
			stat.TXAbortedErrors,
			stat.TXCarrierErrors,
			stat.TXFIFOErrors,
			stat.TXHeartbeatErrors,
			stat.TXWindowErrors,
			stat.TXAbortedRestarts,
			stat.TXBusErrorRestarts,
			stat.Collisions,
			stat.CarrierChanges,
			stat.BusOffRestarts,
			stat.ArbitrationLost,
			stat.ErrorWarning,
			stat.ErrorPassive,
			stat.BusOff,
		)

		if err != nil {
			return fmt.Errorf("failed to append to batch: %w", err)
		}
	}

	err = batch.Send()
	if err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	fmt.Printf("Flushed %d statistics records to ClickHouse\n", len(w.batch))
	w.batch = w.batch[:0] // Clear batch

	return nil
}

// Write queues statistics for writing
func (w *StatsWriter) Write(stat models.SocketCANStats) {
	select {
	case w.batchChan <- stat:
	default:
		fmt.Println("Warning: stats batch channel full, dropping record")
	}
}

// Close closes the statistics writer
func (w *StatsWriter) Close() error {
	w.cancel()
	w.flushTimer.Stop()
	close(w.batchChan)
	return nil
}
