package database

import "can-db-writer/internal/models"

// Writer defines the interface for database writers
type Writer interface {
	// Start begins processing and writing messages
	Start(tableName string)

	// Write queues a message for writing
	Write(msg models.CANMessage)

	// Close closes the database connection and cleans up resources
	Close() error
}
