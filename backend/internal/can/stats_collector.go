package can

import (
	"can-db-writer/internal/models"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StatsCollector collects SocketCAN interface statistics
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

// collect gathers statistics from the CAN interface
func (sc *StatsCollector) collect() {
	stats, err := sc.parseIPStats()
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

// parseIPStats parses the output of 'ip -details -statistics link show'
func (sc *StatsCollector) parseIPStats() (models.SocketCANStats, error) {
	cmd := exec.Command("ip", "-details", "-statistics", "link", "show", sc.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return models.SocketCANStats{}, fmt.Errorf("failed to execute ip command: %w (output: %s)", err, string(output))
	}

	return parseIPOutput(string(output))
}

// parseIPOutput parses the text output from ip command
func parseIPOutput(output string) (models.SocketCANStats, error) {
	stats := models.SocketCANStats{}
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Parse basic interface info (line 0)
		if i == 0 {
			// Example: "3: can0: <NOARP,UP,LOWER_UP,ECHO> mtu 16 qdisc pfifo_fast state UP mode DEFAULT group default qlen 10"
			if matches := regexp.MustCompile(`<([^>]+)>`).FindStringSubmatch(line); len(matches) > 1 {
				if strings.Contains(matches[1], "UP") {
					stats.State = "UP"
				} else {
					stats.State = "DOWN"
				}
			}

			if matches := regexp.MustCompile(`mtu (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.MTU, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`qlen (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.QueueLength, _ = strconv.Atoi(matches[1])
			}
		}

		// Parse CAN-specific parameters
		if strings.Contains(line, "bitrate") {
			// Example: "can state ERROR-ACTIVE (berr-counter tx 0 rx 0) restart-ms 0"
			// or: "bitrate 500000 sample-point 0.875"
			if matches := regexp.MustCompile(`bitrate (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.Bitrate, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`sample-point ([\d.]+)`).FindStringSubmatch(line); len(matches) > 1 {
				samplePoint, _ := strconv.ParseFloat(matches[1], 64)
				stats.SamplePoint = fmt.Sprintf("%.1f%%", samplePoint*100)
			}
		}

		// Parse CAN state and error counters
		if strings.Contains(line, "can state") || strings.Contains(line, "can ") {
			// Example: "can state ERROR-ACTIVE (berr-counter tx 0 rx 0) restart-ms 0"
			if matches := regexp.MustCompile(`state ([A-Z-]+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.BusState = matches[1]
			}

			if matches := regexp.MustCompile(`berr-counter tx (\d+) rx (\d+)`).FindStringSubmatch(line); len(matches) > 2 {
				stats.TXErrorCounter, _ = strconv.Atoi(matches[1])
				stats.RXErrorCounter, _ = strconv.Atoi(matches[2])
			}

			if matches := regexp.MustCompile(`restart-ms (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.RestartMS, _ = strconv.Atoi(matches[1])
			}
		}

		// Parse timing parameters
		if strings.Contains(line, "tq") && strings.Contains(line, "prop-seg") {
			// Example: "tq 125 prop-seg 6 phase-seg1 7 phase-seg2 2 sjw 1"
			if matches := regexp.MustCompile(`tq (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.TimeQuanta, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`prop-seg (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.PropSeg, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`phase-seg1 (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.PhaseSeg1, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`phase-seg2 (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.PhaseSeg2, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`sjw (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.SJW, _ = strconv.Atoi(matches[1])
			}

			if matches := regexp.MustCompile(`brp (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.BRP, _ = strconv.Atoi(matches[1])
			}
		}

		// Parse controller mode
		if strings.Contains(line, "LOOPBACK") {
			stats.ControllerMode = "LOOPBACK"
		} else if strings.Contains(line, "LISTEN-ONLY") {
			stats.ControllerMode = "LISTEN-ONLY"
		}

		// Parse RX statistics
		if strings.Contains(line, "RX:") {
			// Example: "RX: bytes  packets  errors  dropped overrun mcast"
			// Next line: "123456    789      0       0       0       0"
			if i+1 < len(lines) {
				nextLine := strings.Fields(lines[i+1])
				if len(nextLine) >= 6 {
					stats.RXBytes, _ = strconv.ParseUint(nextLine[0], 10, 64)
					stats.RXPackets, _ = strconv.ParseUint(nextLine[1], 10, 64)
					stats.RXErrors, _ = strconv.ParseUint(nextLine[2], 10, 64)
					stats.RXDropped, _ = strconv.ParseUint(nextLine[3], 10, 64)
					stats.RXOverErrors, _ = strconv.ParseUint(nextLine[4], 10, 64)
				}
			}
		}

		// Parse TX statistics
		if strings.Contains(line, "TX:") {
			// Example: "TX: bytes  packets  errors  dropped carrier collsns"
			// Next line: "654321    987      0       0       0       0"
			if i+1 < len(lines) {
				nextLine := strings.Fields(lines[i+1])
				if len(nextLine) >= 6 {
					stats.TXBytes, _ = strconv.ParseUint(nextLine[0], 10, 64)
					stats.TXPackets, _ = strconv.ParseUint(nextLine[1], 10, 64)
					stats.TXErrors, _ = strconv.ParseUint(nextLine[2], 10, 64)
					stats.TXDropped, _ = strconv.ParseUint(nextLine[3], 10, 64)
					stats.TXCarrierErrors, _ = strconv.ParseUint(nextLine[4], 10, 64)
					stats.Collisions, _ = strconv.ParseUint(nextLine[5], 10, 64)
				}
			}
		}

		// Parse CAN-specific error statistics
		if strings.Contains(line, "re-started") {
			// Example: "re-started 0"
			if matches := regexp.MustCompile(`re-started (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.BusOffRestarts, _ = strconv.ParseUint(matches[1], 10, 64)
			}
		}

		if strings.Contains(line, "bus-error") {
			// Example: "bus-error 5"
			if matches := regexp.MustCompile(`bus-error (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.BusErrorCounter, _ = strconv.Atoi(matches[1])
			}
		}

		if strings.Contains(line, "arbitration-lost") {
			if matches := regexp.MustCompile(`arbitration-lost (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.ArbitrationLost, _ = strconv.ParseUint(matches[1], 10, 64)
			}
		}

		if strings.Contains(line, "error-warning") {
			if matches := regexp.MustCompile(`error-warning (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.ErrorWarning, _ = strconv.ParseUint(matches[1], 10, 64)
			}
		}

		if strings.Contains(line, "error-passive") {
			if matches := regexp.MustCompile(`error-passive (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.ErrorPassive, _ = strconv.ParseUint(matches[1], 10, 64)
			}
		}

		if strings.Contains(line, "bus-off") {
			if matches := regexp.MustCompile(`bus-off (\d+)`).FindStringSubmatch(line); len(matches) > 1 {
				stats.BusOff, _ = strconv.ParseUint(matches[1], 10, 64)
			}
		}
	}

	return stats, nil
}
