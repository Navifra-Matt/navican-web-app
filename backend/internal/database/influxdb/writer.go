package influxdb

import (
	"can-db-writer/internal/models"
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Writer handles writing CAN messages to InfluxDB
type Writer struct {
	client     influxdb2.Client
	writeAPI   api.WriteAPI
	batchSize  int
	batch      []models.CANMessage
	batchChan  chan models.CANMessage
	ctx        context.Context
	cancel     context.CancelFunc
	flushTimer *time.Ticker
	bucket     string
	org        string
}

// New creates a new InfluxDB writer
func New(config Config, batchSize int) (*Writer, error) {
	// Create InfluxDB client
	client := influxdb2.NewClient(config.URL, config.Token)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		return nil, fmt.Errorf("InfluxDB health check failed: %s", health.Message)
	}

	ctx, cancel = context.WithCancel(context.Background())

	// Create write API
	writeAPI := client.WriteAPI(config.Org, config.Bucket)

	writer := &Writer{
		client:     client,
		writeAPI:   writeAPI,
		batchSize:  batchSize,
		batch:      make([]models.CANMessage, 0, batchSize),
		batchChan:  make(chan models.CANMessage, batchSize*2),
		ctx:        ctx,
		cancel:     cancel,
		flushTimer: time.NewTicker(1 * time.Second), // Flush every second
		bucket:     config.Bucket,
		org:        config.Org,
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

	for _, msg := range w.batch {
		// Create point with measurement name "can_messages"
		point := write.NewPoint(
			"can_messages",
			map[string]string{
				"interface": msg.Interface,
				"can_id":    fmt.Sprintf("0x%X", msg.Frame.ID),
			},
			map[string]any{
				"can_id_decimal": msg.Frame.ID,
				"dlc":            msg.Frame.DLC,
				"data_0":         msg.Frame.Data[0],
				"data_1":         msg.Frame.Data[1],
				"data_2":         msg.Frame.Data[2],
				"data_3":         msg.Frame.Data[3],
				"data_4":         msg.Frame.Data[4],
				"data_5":         msg.Frame.Data[5],
				"data_6":         msg.Frame.Data[6],
				"data_7":         msg.Frame.Data[7],
				"data_hex":       fmt.Sprintf("%02X %02X %02X %02X %02X %02X %02X %02X",
					msg.Frame.Data[0], msg.Frame.Data[1], msg.Frame.Data[2], msg.Frame.Data[3],
					msg.Frame.Data[4], msg.Frame.Data[5], msg.Frame.Data[6], msg.Frame.Data[7]),
			},
			msg.Timestamp,
		)

		// Write point
		w.writeAPI.WritePoint(point)
	}

	// Flush to ensure data is written
	w.writeAPI.Flush()

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
	w.writeAPI.Flush()

	if w.client != nil {
		w.client.Close()
	}
	return nil
}
