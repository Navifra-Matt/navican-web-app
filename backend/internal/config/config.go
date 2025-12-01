package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// CAN Interface
	CANInterface   string
	CANFilters     []uint32
	StatsInterval  int

	// ClickHouse
	ClickHouseHost     string
	ClickHousePort     int
	ClickHouseDatabase string
	ClickHouseUsername string
	ClickHousePassword string
	ClickHouseTable    string
	ClickHouseStatsTable string

	// InfluxDB
	InfluxDBURL    string
	InfluxDBToken  string
	InfluxDBOrg    string
	InfluxDBBucket string

	// General
	BatchSize int
	APIPort   int
}

// LoadConfig loads configuration from .env file
func LoadConfig(envFile string) (*Config, error) {
	// Set default values
	config := &Config{
		CANInterface:         "vcan0",
		StatsInterval:        10,
		ClickHouseHost:       "localhost",
		ClickHousePort:       9000,
		ClickHouseDatabase:   "default",
		ClickHouseUsername:   "default",
		ClickHousePassword:   "",
		ClickHouseTable:      "can_messages",
		ClickHouseStatsTable: "can_interface_stats",
		InfluxDBURL:          "http://localhost:8086",
		InfluxDBToken:        "",
		InfluxDBOrg:          "my-org",
		InfluxDBBucket:       "can_messages",
		BatchSize:            1000,
		APIPort:              8080,
	}

	// Try to load .env file
	if envFile == "" {
		envFile = ".env"
	}

	file, err := os.Open(envFile)
	if err != nil {
		// If .env file doesn't exist, return default config
		if os.IsNotExist(err) {
			fmt.Printf("No .env file found at %s, using default configuration\n", envFile)
			return config, nil
		}
		return nil, fmt.Errorf("error opening .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		// Set configuration values
		switch key {
		case "CAN_INTERFACE":
			config.CANInterface = value
		case "CAN_FILTERS":
			config.CANFilters = parseFilters(value)
		case "STATS_INTERVAL":
			config.StatsInterval, _ = strconv.Atoi(value)
		case "CLICKHOUSE_HOST":
			config.ClickHouseHost = value
		case "CLICKHOUSE_PORT":
			config.ClickHousePort, _ = strconv.Atoi(value)
		case "CLICKHOUSE_DATABASE":
			config.ClickHouseDatabase = value
		case "CLICKHOUSE_USERNAME":
			config.ClickHouseUsername = value
		case "CLICKHOUSE_PASSWORD":
			config.ClickHousePassword = value
		case "CLICKHOUSE_TABLE":
			config.ClickHouseTable = value
		case "CLICKHOUSE_STATS_TABLE":
			config.ClickHouseStatsTable = value
		case "INFLUXDB_URL":
			config.InfluxDBURL = value
		case "INFLUXDB_TOKEN":
			config.InfluxDBToken = value
		case "INFLUXDB_ORG":
			config.InfluxDBOrg = value
		case "INFLUXDB_BUCKET":
			config.InfluxDBBucket = value
		case "BATCH_SIZE":
			config.BatchSize, _ = strconv.Atoi(value)
		case "API_PORT":
			config.APIPort, _ = strconv.Atoi(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return config, nil
}

// parseFilters parses comma-separated CAN IDs
func parseFilters(filterStr string) []uint32 {
	if filterStr == "" {
		return nil
	}

	parts := strings.Split(filterStr, ",")
	filters := make([]uint32, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var id uint32
		_, err := fmt.Sscanf(part, "%x", &id)
		if err != nil {
			continue
		}

		filters = append(filters, id)
	}

	return filters
}
