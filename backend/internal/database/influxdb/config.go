package influxdb

// Config holds InfluxDB connection configuration
type Config struct {
	URL      string
	Token    string
	Database string // InfluxDB v3 uses Database instead of Org/Bucket
}
