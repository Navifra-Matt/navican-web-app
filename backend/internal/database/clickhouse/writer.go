package clickhouse

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Writer handles writing CAN messages to ClickHouse
type Writer struct {
	conn       driver.Conn
	config     Config
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
		config:     config,
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
		TTL timestamp + INTERVAL 1 MONTH
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

// ExportFormat represents the export file format
type ExportFormat string

const (
	FormatParquet ExportFormat = "Parquet"
	FormatIceberg ExportFormat = "Iceberg"
)

// ExportOptions contains options for exporting data
type ExportOptions struct {
	Format      ExportFormat
	StartTime   time.Time
	EndTime     time.Time
	OutputPath  string
	Compression string // snappy, lz4, brotli, zstd, gzip, none (uncompressed) - default: zstd
}

// ExportToParquet exports data to Parquet format
func (w *Writer) ExportToParquet(tableName string, opts ExportOptions) error {
	if opts.Compression == "" {
		opts.Compression = "zstd"
	}

	// Ensure output directory exists
	dir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build query with time range filter
	query := fmt.Sprintf(`
		SELECT
			timestamp,
			interface,
			can_id,
			data
		FROM %s
		WHERE timestamp >= '%s' AND timestamp < '%s'
		ORDER BY timestamp
		INTO OUTFILE '%s'
		FORMAT Parquet
		SETTINGS output_format_parquet_compression_method='%s'
	`,
		tableName,
		opts.StartTime.Format("2006-01-02 15:04:05"),
		opts.EndTime.Format("2006-01-02 15:04:05"),
		opts.OutputPath,
		opts.Compression,
	)

	if err := w.conn.Exec(context.Background(), query); err != nil {
		return fmt.Errorf("failed to export to Parquet: %w", err)
	}

	fmt.Printf("Successfully exported data to Parquet: %s\n", opts.OutputPath)
	return nil
}

// ExportToIceberg exports data to Iceberg format
func (w *Writer) ExportToIceberg(tableName string, opts ExportOptions) error {
	if opts.Compression == "" {
		opts.Compression = "zstd"
	}

	// Ensure output directory exists
	dir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build query with time range filter
	query := fmt.Sprintf(`
		SELECT
			timestamp,
			interface,
			can_id,
			data
		FROM %s
		WHERE timestamp >= '%s' AND timestamp < '%s'
		ORDER BY timestamp
		INTO OUTFILE '%s'
		FORMAT Iceberg
		SETTINGS output_format_parquet_compression_method='%s'
	`,
		tableName,
		opts.StartTime.Format("2006-01-02 15:04:05"),
		opts.EndTime.Format("2006-01-02 15:04:05"),
		opts.OutputPath,
		opts.Compression,
	)

	if err := w.conn.Exec(context.Background(), query); err != nil {
		return fmt.Errorf("failed to export to Iceberg: %w", err)
	}

	fmt.Printf("Successfully exported data to Iceberg: %s\n", opts.OutputPath)
	return nil
}

// ExportToWriter exports data directly to an io.Writer in the specified format
// This is used for streaming exports via HTTP using ClickHouse native format support
func (w *Writer) ExportToWriter(writer io.Writer, tableName string, opts ExportOptions) error {
	if opts.Compression == "" {
		opts.Compression = "zstd"
	}

	// Determine format and settings
	var formatStr string
	var settings string

	switch opts.Format {
	case FormatIceberg:
		formatStr = "Iceberg"
		// Iceberg format settings
		settings = fmt.Sprintf("SETTINGS output_format_parquet_compression_method='%s'", opts.Compression)
	case FormatParquet:
		fallthrough
	default:
		formatStr = "Parquet"
		settings = fmt.Sprintf("SETTINGS output_format_parquet_compression_method='%s'", opts.Compression)
	}

	// Build query with ClickHouse's native format output
	query := fmt.Sprintf(`
		SELECT
			timestamp,
			interface,
			can_id,
			data
		FROM %s
		WHERE timestamp >= '%s' AND timestamp < '%s'
		ORDER BY timestamp
		FORMAT %s
		%s
	`,
		tableName,
		opts.StartTime.Format("2006-01-02 15:04:05"),
		opts.EndTime.Format("2006-01-02 15:04:05"),
		formatStr,
		settings,
	)

	// Use ClickHouse HTTP interface to get format directly
	httpURL := fmt.Sprintf("http://%s:%d/", w.config.Host, 8123) // ClickHouse HTTP port is typically 8123

	// Create HTTP request with query
	params := url.Values{}
	params.Set("query", query)
	params.Set("database", w.config.Database)

	// Add authentication if needed
	if w.config.Username != "" {
		params.Set("user", w.config.Username)
		params.Set("password", w.config.Password)
	}

	// Make HTTP GET request
	fullURL := httpURL + "?" + params.Encode()
	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ClickHouse HTTP query failed with status %d", resp.StatusCode)
	}

	// Copy the data from response to writer
	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy %s data: %w", formatStr, err)
	}

	fmt.Printf("Successfully exported %d bytes to %s format\n", written, formatStr)
	return nil
}
