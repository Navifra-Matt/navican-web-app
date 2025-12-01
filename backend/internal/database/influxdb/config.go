package influxdb

// Config holds InfluxDB connection configuration
type Config struct {
	URL    string
	Token  string
	Org    string
	Bucket string
}
