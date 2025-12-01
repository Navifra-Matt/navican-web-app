package models

import "time"

// CANFrame represents a CAN 2.0 frame
type CANFrame struct {
	ID   uint32
	DLC  uint8
	Data [8]byte
}

// CANMessage includes the CAN frame and timestamp
type CANMessage struct {
	Frame     CANFrame
	Timestamp time.Time
	Interface string
}

// CANMessageResponse represents a CAN message in API response
type CANMessageResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Interface string    `json:"interface"`
	CANID     uint32    `json:"can_id"`
	CANIDHex  string    `json:"can_id_hex"`
	DLC       uint8     `json:"dlc"`
	Data      []uint8   `json:"data"`
	DataHex   string    `json:"data_hex"`
}
