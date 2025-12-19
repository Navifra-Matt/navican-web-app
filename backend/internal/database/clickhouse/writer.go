package clickhouse

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Writer handles writing CAN messages to ClickHouse
type Writer struct {
	conn       driver.Conn
	batchSize  int
	batch      []models.CANMessage
	batchChan  chan models.CANMessage
	ctx        context.Context
	cancel     context.CancelFunc
	flushTimer *time.Ticker
}

// New creates a new ClickHouse writer
func New(config Config, batchSize int) (*Writer, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// Create table if not exists
	err = createTable(conn, config.Table)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	writer := &Writer{
		conn:       conn,
		batchSize:  batchSize,
		batch:      make([]models.CANMessage, 0, batchSize),
		batchChan:  make(chan models.CANMessage, batchSize*2),
		ctx:        ctx,
		cancel:     cancel,
		flushTimer: time.NewTicker(1 * time.Second), // Flush every second
	}

	return writer, nil
}

// createTable creates the CAN messages table in ClickHouse
func createTable(conn driver.Conn, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			timestamp DateTime64(6),
			interface String,
			can_id UInt32,
			data Array(UInt8)
		) ENGINE = MergeTree()
		ORDER BY (timestamp, can_id)
		PARTITION BY toYYYYMMDD(timestamp)
		SETTINGS index_granularity = 8192
	`, tableName)

	return conn.Exec(context.Background(), query)
}

// Start begins processing and writing messages
func (w *Writer) Start(tableName string) {
	go w.writeLoop(tableName)
}

// writeLoop processes messages and writes them in batches
func (w *Writer) writeLoop(tableName string) {
	for {
		select {
		case <-w.ctx.Done():
			// Flush remaining messages before exiting
			if len(w.batch) > 0 {
				w.flush(tableName)
			}
			return

		case msg := <-w.batchChan:
			w.batch = append(w.batch, msg)
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
func (w *Writer) flush(tableName string) error {
	if len(w.batch) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(w.ctx, fmt.Sprintf("INSERT INTO %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, msg := range w.batch {
		err = batch.Append(
			msg.Timestamp,
			msg.Interface,
			msg.Frame.ID,
			msg.Frame.Data[:],
		)

		if err != nil {
			return fmt.Errorf("failed to append to batch: %w", err)
		}
	}

	err = batch.Send()
	if err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	fmt.Printf("Flushed %d messages to ClickHouse\n", len(w.batch))
	w.batch = w.batch[:0] // Clear batch

	return nil
}

// Write queues a message for writing
func (w *Writer) Write(msg models.CANMessage) {
	select {
	case w.batchChan <- msg:
	default:
		fmt.Println("Warning: batch channel full, dropping message")
	}
}

// Close closes the ClickHouse connection
func (w *Writer) Close() error {
	w.cancel()
	w.flushTimer.Stop()
	close(w.batchChan)

	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// GetConn returns the underlying ClickHouse connection
func (w *Writer) GetConn() driver.Conn {
	return w.conn
}
