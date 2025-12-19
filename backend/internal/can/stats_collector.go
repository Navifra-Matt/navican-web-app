package can

import (
	"can-db-writer/internal/models"
	"fmt"
	"time"

	"github.com/vishvananda/netlink"
)

// StatsCollector collects SocketCAN interface statistics using netlink
type StatsCollector struct {
	interfaceName string
	interval      time.Duration
	statsChan     chan models.SocketCANStats
	stopChan      chan struct{}
}

// NewStatsCollector creates a new statistics collector
func NewStatsCollector(interfaceName string, interval time.Duration) *StatsCollector {
	return &StatsCollector{
		interfaceName: interfaceName,
		interval:      interval,
		statsChan:     make(chan models.SocketCANStats, 10),
		stopChan:      make(chan struct{}),
	}
}

// Start begins collecting statistics
func (sc *StatsCollector) Start() {
	go sc.collectLoop()
}

// Stop stops the statistics collector
func (sc *StatsCollector) Stop() {
	close(sc.stopChan)
	close(sc.statsChan)
}

// GetStatsChannel returns the channel for receiving statistics
func (sc *StatsCollector) GetStatsChannel() <-chan models.SocketCANStats {
	return sc.statsChan
}

// collectLoop periodically collects statistics
func (sc *StatsCollector) collectLoop() {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	// Collect immediately on start
	sc.collect()

	for {
		select {
		case <-ticker.C:
			sc.collect()
		case <-sc.stopChan:
			return
		}
	}
}

// collect gathers statistics from the CAN interface using netlink
func (sc *StatsCollector) collect() {
	stats, err := sc.getNetlinkStats()
	if err != nil {
		fmt.Printf("Failed to collect stats for %s: %v\n", sc.interfaceName, err)
		return
	}

	stats.Timestamp = time.Now()
	stats.Interface = sc.interfaceName

	select {
	case sc.statsChan <- stats:
	default:
		fmt.Println("Warning: stats channel full, dropping statistics")
	}
}

// getNetlinkStats retrieves statistics using netlink library
func (sc *StatsCollector) getNetlinkStats() (models.SocketCANStats, error) {
	stats := models.SocketCANStats{}

	// Get link by name
	link, err := netlink.LinkByName(sc.interfaceName)
	if err != nil {
		return stats, fmt.Errorf("failed to get link %s: %w", sc.interfaceName, err)
	}

	attrs := link.Attrs()
	if attrs == nil {
		return stats, fmt.Errorf("link attributes are nil")
	}

	// Basic interface info
	stats.MTU = attrs.MTU
	stats.QueueLength = attrs.TxQLen

	// Interface state
	if attrs.OperState == netlink.OperUp {
		stats.State = "UP"
	} else {
		stats.State = "DOWN"
	}

	// Link statistics
	if linkStats := attrs.Statistics; linkStats != nil {
		// RX statistics
		stats.RXPackets = linkStats.RxPackets
		stats.RXBytes = linkStats.RxBytes
		stats.RXErrors = linkStats.RxErrors
		stats.RXDropped = linkStats.RxDropped
		stats.RXOverErrors = linkStats.RxOverErrors
		stats.RXCRCErrors = linkStats.RxCrcErrors
		stats.RXFrameErrors = linkStats.RxFrameErrors
		stats.RXFIFOErrors = linkStats.RxFifoErrors
		stats.RXMissed = linkStats.RxMissedErrors

		// TX statistics
		stats.TXPackets = linkStats.TxPackets
		stats.TXBytes = linkStats.TxBytes
		stats.TXErrors = linkStats.TxErrors
		stats.TXDropped = linkStats.TxDropped
		stats.TXAbortedErrors = linkStats.TxAbortedErrors
		stats.TXCarrierErrors = linkStats.TxCarrierErrors
		stats.TXFIFOErrors = linkStats.TxFifoErrors
		stats.TXHeartbeatErrors = linkStats.TxHeartbeatErrors
		stats.TXWindowErrors = linkStats.TxWindowErrors

		// Other statistics
		stats.Collisions = linkStats.Collisions
	}

	// CAN-specific parameters (requires type assertion to *netlink.Can)
	if canLink, ok := link.(*netlink.Can); ok {
		stats.Bitrate = int(canLink.BitRate)
		stats.RestartMS = int(canLink.RestartMs)

		// Error counters
		stats.TXErrorCounter = int(canLink.TxError)
		stats.RXErrorCounter = int(canLink.RxError)

		// CAN state (custom mapping based on state value)
		switch canLink.State {
		case 0:
			stats.BusState = "ERROR-ACTIVE"
		case 1:
			stats.BusState = "ERROR-WARNING"
		case 2:
			stats.BusState = "ERROR-PASSIVE"
		case 3:
			stats.BusState = "BUS-OFF"
		case 4:
			stats.BusState = "STOPPED"
		case 5:
			stats.BusState = "SLEEPING"
		default:
			stats.BusState = fmt.Sprintf("UNKNOWN(%d)", canLink.State)
		}

		// Control mode flags
		const (
			CAN_CTRLMODE_LOOPBACK    = 0x01
			CAN_CTRLMODE_LISTENONLY  = 0x02
		)

		if canLink.Flags&CAN_CTRLMODE_LOOPBACK != 0 {
			stats.ControllerMode = "LOOPBACK"
		} else if canLink.Flags&CAN_CTRLMODE_LISTENONLY != 0 {
			stats.ControllerMode = "LISTEN-ONLY"
		} else {
			stats.ControllerMode = "NORMAL"
		}

		// Bit timing parameters
		stats.BRP = int(canLink.BitRatePreScaler)
		stats.PropSeg = int(canLink.PropagationSegment)
		stats.PhaseSeg1 = int(canLink.PhaseSegment1)
		stats.PhaseSeg2 = int(canLink.PhaseSegment2)
		stats.SJW = int(canLink.SyncJumpWidth)
		stats.TimeQuanta = int(canLink.TimeQuanta)

		// Sample point
		if canLink.SamplePoint > 0 {
			// SamplePoint is already in percentage * 10 (e.g., 875 for 87.5%)
			stats.SamplePoint = fmt.Sprintf("%.1f%%", float64(canLink.SamplePoint)/10.0)
		} else if stats.PropSeg > 0 || stats.PhaseSeg1 > 0 || stats.PhaseSeg2 > 0 {
			// Calculate if not provided
			totalTq := 1 + stats.PropSeg + stats.PhaseSeg1 + stats.PhaseSeg2
			if totalTq > 0 {
				samplePointTq := 1 + stats.PropSeg + stats.PhaseSeg1
				samplePoint := float64(samplePointTq) / float64(totalTq) * 100.0
				stats.SamplePoint = fmt.Sprintf("%.1f%%", samplePoint)
			}
		}
	}

	return stats, nil
}
