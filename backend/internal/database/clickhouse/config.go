package clickhouse

// Config holds ClickHouse connection configuration
type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Table    string
}
