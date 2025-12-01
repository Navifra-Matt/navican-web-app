package models

import "time"

// SocketCANStats represents statistics from a SocketCAN interface
type SocketCANStats struct {
	Interface string    `json:"interface"`
	Timestamp time.Time `json:"timestamp"`

	// Interface state
	State       string `json:"state"`        // UP, DOWN, etc.
	MTU         int    `json:"mtu"`          // Maximum Transmission Unit
	QueueLength int    `json:"queue_length"` // TX queue length

	// CAN-specific parameters
	Bitrate          int    `json:"bitrate"`           // Bitrate in bps
	SamplePoint      string `json:"sample_point"`      // Sample point (e.g., "87.5%")
	TimeQuanta       int    `json:"time_quanta"`       // Time quanta in ns
	PropSeg          int    `json:"prop_seg"`          // Propagation segment
	PhaseSeg1        int    `json:"phase_seg1"`        // Phase segment 1
	PhaseSeg2        int    `json:"phase_seg2"`        // Phase segment 2
	SJW              int    `json:"sjw"`               // Synchronization Jump Width
	BRP              int    `json:"brp"`               // Bit Rate Prescaler
	RestartMS        int    `json:"restart_ms"`        // Auto-restart delay in ms
	ControllerMode   string `json:"controller_mode"`   // Controller mode (e.g., "LOOPBACK")
	BusState         string `json:"bus_state"`         // Bus state (ERROR-ACTIVE, ERROR-PASSIVE, BUS-OFF)
	BusErrorCounter  int    `json:"bus_error_counter"` // Bus error counter
	RXErrorCounter   int    `json:"rx_error_counter"`  // RX error counter
	TXErrorCounter   int    `json:"tx_error_counter"`  // TX error counter

	// RX statistics
	RXPackets     uint64 `json:"rx_packets"`      // Total received packets
	RXBytes       uint64 `json:"rx_bytes"`        // Total received bytes
	RXErrors      uint64 `json:"rx_errors"`       // Total receive errors
	RXDropped     uint64 `json:"rx_dropped"`      // Dropped packets on receive
	RXOverErrors  uint64 `json:"rx_over_errors"`  // Receiver ring buff overflow
	RXCRCErrors   uint64 `json:"rx_crc_errors"`   // CRC errors
	RXFrameErrors uint64 `json:"rx_frame_errors"` // Frame alignment errors
	RXFIFOErrors  uint64 `json:"rx_fifo_errors"`  // Recv FIFO errors
	RXMissed      uint64 `json:"rx_missed"`       // Missed packets

	// TX statistics
	TXPackets          uint64 `json:"tx_packets"`            // Total transmitted packets
	TXBytes            uint64 `json:"tx_bytes"`              // Total transmitted bytes
	TXErrors           uint64 `json:"tx_errors"`             // Total transmit errors
	TXDropped          uint64 `json:"tx_dropped"`            // Dropped packets on transmit
	TXAbortedErrors    uint64 `json:"tx_aborted_errors"`     // Transmit aborted errors
	TXCarrierErrors    uint64 `json:"tx_carrier_errors"`     // Carrier errors
	TXFIFOErrors       uint64 `json:"tx_fifo_errors"`        // Transmit FIFO errors
	TXHeartbeatErrors  uint64 `json:"tx_heartbeat_errors"`   // Heartbeat errors
	TXWindowErrors     uint64 `json:"tx_window_errors"`      // Window errors
	TXAbortedRestarts  uint64 `json:"tx_aborted_restarts"`   // Aborted restarts (CAN specific)
	TXBusErrorRestarts uint64 `json:"tx_bus_error_restarts"` // Bus error restarts (CAN specific)

	// Additional statistics
	Collisions      uint64 `json:"collisions"`       // Collisions
	CarrierChanges  uint64 `json:"carrier_changes"`  // Number of carrier changes
	BusOffRestarts  uint64 `json:"bus_off_restarts"` // Bus-off restarts (CAN specific)
	ArbitrationLost uint64 `json:"arbitration_lost"` // Arbitration lost count (CAN specific)
	ErrorWarning    uint64 `json:"error_warning"`    // Error warning state entries (CAN specific)
	ErrorPassive    uint64 `json:"error_passive"`    // Error passive state entries (CAN specific)
	BusOff          uint64 `json:"bus_off"`          // Bus-off state entries (CAN specific)
}
