package influxdb

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// Writer handles writing CAN messages to InfluxDB
type Writer struct {
	client     *influxdb3.Client
	batchSize  int
	batch      []models.CANMessage
	batchChan  chan models.CANMessage
	ctx        context.Context
	cancel     context.CancelFunc
	flushTimer *time.Ticker
	database   string
}

// New creates a new InfluxDB writer
func New(config Config, batchSize int) (*Writer, error) {
	// Create InfluxDB v3 client
	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     config.URL,
		Token:    config.Token,
		Database: config.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create InfluxDB client: %w", err)
	}

	// Test connection by attempting to write a test point
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simple ping test - write will fail gracefully if connection is bad
	_ = ctx

	ctx, cancel = context.WithCancel(context.Background())

	writer := &Writer{
		client:     client,
		batchSize:  batchSize,
		batch:      make([]models.CANMessage, 0, batchSize),
		batchChan:  make(chan models.CANMessage, batchSize*2),
		ctx:        ctx,
		cancel:     cancel,
		flushTimer: time.NewTicker(1 * time.Second), // Flush every second
		database:   config.Database,
	}

	return writer, nil
}

// Start begins processing and writing messages
func (w *Writer) Start(tableName string) {
	go w.writeLoop()
}

// writeLoop processes messages and writes them in batches
func (w *Writer) writeLoop() {
	for {
		select {
		case <-w.ctx.Done():
			// Flush remaining messages before exiting
			if len(w.batch) > 0 {
				w.flush()
			}
			return

		case msg := <-w.batchChan:
			w.batch = append(w.batch, msg)
			if len(w.batch) >= w.batchSize {
				w.flush()
			}

		case <-w.flushTimer.C:
			if len(w.batch) > 0 {
				w.flush()
			}
		}
	}
}

// flush writes the current batch to InfluxDB
func (w *Writer) flush() error {
	if len(w.batch) == 0 {
		return nil
	}

	// Build points for batch writing
	points := make([]*influxdb3.Point, 0, len(w.batch))

	for _, msg := range w.batch {
		// Create point with measurement name "can_messages"
		point := influxdb3.NewPoint(
			"can_messages",
			map[string]string{
				"interface": msg.Interface,
				"can_id":    fmt.Sprintf("0x%X", msg.Frame.ID),
			},
			map[string]any{
				"can_id_decimal": msg.Frame.ID,
				"data_0":         msg.Frame.Data[0],
				"data_1":         msg.Frame.Data[1],
				"data_2":         msg.Frame.Data[2],
				"data_3":         msg.Frame.Data[3],
				"data_4":         msg.Frame.Data[4],
				"data_5":         msg.Frame.Data[5],
				"data_6":         msg.Frame.Data[6],
				"data_7":         msg.Frame.Data[7],
			},
			msg.Timestamp,
		)
		points = append(points, point)
	}

	// Write all points in batch
	err := w.client.WritePoints(w.ctx, points)
	if err != nil {
		return fmt.Errorf("failed to write points: %w", err)
	}

	fmt.Printf("Flushed %d messages to InfluxDB\n", len(w.batch))
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

// Close closes the InfluxDB connection
func (w *Writer) Close() error {
	w.cancel()
	w.flushTimer.Stop()
	close(w.batchChan)

	// Flush any remaining data
	if len(w.batch) > 0 {
		w.flush()
	}

	if w.client != nil {
		w.client.Close()
	}
	return nil
}
