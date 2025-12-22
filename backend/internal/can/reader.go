package can

import (
	"can-db-writer/internal/models"
	"encoding/binary"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	CAN_RAW        = 1
	SOL_CAN_RAW    = 101
	CAN_RAW_FILTER = 1
)

// Reader handles reading from SocketCAN
type Reader struct {
	socket    int
	ifname    string
	msgChan   chan models.CANMessage
	errorChan chan error
}

// NewReader creates a new CAN reader for the specified interface
func NewReader(ifname string) (*Reader, error) {
	// Create a CAN socket
	socket, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, CAN_RAW)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAN socket: %w", err)
	}

	// Get interface index
	ifreq, err := unix.NewIfreq(ifname)
	if err != nil {
		unix.Close(socket)
		return nil, fmt.Errorf("failed to create ifreq: %w", err)
	}

	err = unix.IoctlIfreq(socket, unix.SIOCGIFINDEX, ifreq)
	if err != nil {
		unix.Close(socket)
		return nil, fmt.Errorf("failed to get interface index: %w", err)
	}

	ifindex := ifreq.Uint32()

	// Bind socket to CAN interface
	addr := &unix.SockaddrCAN{
		Ifindex: int(ifindex),
	}

	err = unix.Bind(socket, addr)
	if err != nil {
		unix.Close(socket)
		return nil, fmt.Errorf("failed to bind socket: %w", err)
	}

	return &Reader{
		socket:    socket,
		ifname:    ifname,
		msgChan:   make(chan models.CANMessage, 1000),
		errorChan: make(chan error, 10),
	}, nil
}

// Start begins reading CAN frames
func (r *Reader) Start() {
	go r.readLoop()
}

// readLoop continuously reads CAN frames from the socket
func (r *Reader) readLoop() {
	buf := make([]byte, 16) // CAN frame is 16 bytes

	for {
		n, err := unix.Read(r.socket, buf)
		if err != nil {
			r.errorChan <- fmt.Errorf("read error: %w", err)
			continue
		}

		if n < 16 {
			r.errorChan <- fmt.Errorf("incomplete CAN frame received: %d bytes", n)
			continue
		}

		// Parse CAN frame
		frame := models.CANFrame{
			ID:  binary.LittleEndian.Uint32(buf[0:4]),
			DLC: buf[4],
		}
		copy(frame.Data[:], buf[8:16])

		msg := models.CANMessage{
			Frame:     frame,
			Timestamp: time.Now().UTC(),
			Interface: r.ifname,
		}

		select {
		case r.msgChan <- msg:
		default:
			r.errorChan <- fmt.Errorf("message channel full, dropping frame")
		}
	}
}

// GetMessageChannel returns the channel for receiving CAN messages
func (r *Reader) GetMessageChannel() <-chan models.CANMessage {
	return r.msgChan
}

// GetErrorChannel returns the channel for receiving errors
func (r *Reader) GetErrorChannel() <-chan error {
	return r.errorChan
}

// Close closes the CAN socket
func (r *Reader) Close() error {
	close(r.msgChan)
	close(r.errorChan)
	return unix.Close(r.socket)
}

// Helper function to check if socket FD is valid
func isFDValid(fd int) bool {
	var stat syscall.Stat_t
	err := syscall.Fstat(fd, &stat)
	return err == nil
}

// SetFilter sets CAN ID filters (optional)
func (r *Reader) SetFilter(filters []uint32) error {
	if len(filters) == 0 {
		return nil
	}

	// CAN filter structure: 8 bytes (4 for ID, 4 for mask)
	filterBuf := make([]byte, len(filters)*8)
	for i, id := range filters {
		offset := i * 8
		binary.LittleEndian.PutUint32(filterBuf[offset:], id)
		binary.LittleEndian.PutUint32(filterBuf[offset+4:], 0xFFFFFFFF) // exact match
	}

	_, _, errno := syscall.Syscall6(
		syscall.SYS_SETSOCKOPT,
		uintptr(r.socket),
		uintptr(SOL_CAN_RAW),
		uintptr(CAN_RAW_FILTER),
		uintptr(unsafe.Pointer(&filterBuf[0])),
		uintptr(len(filterBuf)),
		0,
	)

	if errno != 0 {
		return fmt.Errorf("failed to set filter: %v", errno)
	}

	return nil
}
