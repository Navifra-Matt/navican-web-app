package main

import (
	"can-db-writer/internal/can"
	"can-db-writer/internal/config"
	"can-db-writer/internal/database/clickhouse"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Command line flag for config file
	envFile := flag.String("env", ".env", "Path to .env configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting CAN to Database bridge...")
	log.Printf("CAN Interface: %s", cfg.CANInterface)
	log.Printf("ClickHouse: %s:%d/%s.%s", cfg.ClickHouseHost, cfg.ClickHousePort, cfg.ClickHouseDatabase, cfg.ClickHouseTable)

	// Create CAN reader
	canReader, err := can.NewReader(cfg.CANInterface)
	if err != nil {
		log.Fatalf("Failed to create CAN reader: %v", err)
	}
	defer canReader.Close()

	// Set filters if provided
	if len(cfg.CANFilters) > 0 {
		err = canReader.SetFilter(cfg.CANFilters)
		if err != nil {
			log.Printf("Warning: Failed to set filters: %v", err)
		} else {
			log.Printf("Applied CAN ID filters: %v", cfg.CANFilters)
		}
	}

	// Create ClickHouse writer
	chConfig := clickhouse.Config{
		Host:     cfg.ClickHouseHost,
		Port:     cfg.ClickHousePort,
		Database: cfg.ClickHouseDatabase,
		Username: cfg.ClickHouseUsername,
		Password: cfg.ClickHousePassword,
		Table:    cfg.ClickHouseTable,
	}

	chWriter, err := clickhouse.New(chConfig, cfg.BatchSize)
	if err != nil {
		log.Fatalf("Failed to create ClickHouse writer: %v", err)
	}
	defer chWriter.Close()

	// Create statistics table and writer
	err = clickhouse.CreateStatsTable(chWriter.GetConn(), cfg.ClickHouseStatsTable)
	if err != nil {
		log.Fatalf("Failed to create statistics table: %v", err)
	}

	statsWriter := clickhouse.NewStatsWriter(chWriter.GetConn(), cfg.BatchSize/10)
	defer statsWriter.Close()

	// Create and start statistics collector
	statsCollector := can.NewStatsCollector(cfg.CANInterface, time.Duration(cfg.StatsInterval)*time.Second)
	statsCollector.Start()
	defer statsCollector.Stop()

	// Start readers and writers
	canReader.Start()
	chWriter.Start(cfg.ClickHouseTable)
	statsWriter.Start(cfg.ClickHouseStatsTable)

	log.Println("Bridge started successfully. Press Ctrl+C to stop.")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Statistics
	var messageCount uint64
	var errorCount uint64

	// Message processing loop
	go func() {
		for {
			select {
			case msg := <-canReader.GetMessageChannel():
				messageCount++
			// Write to ClickHouse
			chWriter.Write(msg)

				// Log every 1000 messages
				if messageCount%1000 == 0 {
					log.Printf("Processed %d messages (errors: %d)", messageCount, errorCount)
				}

			case err := <-canReader.GetErrorChannel():
				errorCount++
				log.Printf("CAN error: %v", err)
			}
		}
	}()

	// Statistics collection loop
	go func() {
		for stat := range statsCollector.GetStatsChannel() {
			statsWriter.Write(stat)
			log.Printf("Collected statistics for %s: RX packets=%d, TX packets=%d, Bus state=%s",
				stat.Interface, stat.RXPackets, stat.TXPackets, stat.BusState)
		}
	}()

	// Wait for termination signal
	<-sigChan
	log.Println("\nShutting down...")
	log.Printf("Final statistics: %d messages processed, %d errors", messageCount, errorCount)
}
