package models

import "time"

// QueryParams represents common query parameters
type QueryParams struct {
	StartTime *time.Time
	EndTime   *time.Time
	CANID     *uint32
	Interface string
	Limit     int
	Offset    int
}
