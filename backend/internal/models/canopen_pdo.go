package models

import (
	"encoding/binary"
)

// PDOFieldType represents the data type of a PDO field
type PDOFieldType string

const (
	FieldTypeInt8   PDOFieldType = "int8"
	FieldTypeUint8  PDOFieldType = "uint8"
	FieldTypeInt16  PDOFieldType = "int16"
	FieldTypeUint16 PDOFieldType = "uint16"
	FieldTypeInt32  PDOFieldType = "int32"
	FieldTypeUint32 PDOFieldType = "uint32"
)

// PDOField defines a single field in a PDO message
type PDOField struct {
	Name       string       `json:"name"`        // Field name (e.g., "statusword")
	Type       PDOFieldType `json:"type"`        // Data type
	ByteOffset int          `json:"byte_offset"` // Starting byte position (0-7)
	ByteLength int          `json:"byte_length"` // Number of bytes (1, 2, or 4)
}

// PDOMapping defines the complete mapping for a PDO type
type PDOMapping struct {
	PDONumber   int        `json:"pdo_number"`   // PDO number (1-4)
	Direction   string     `json:"direction"`    // "TX" or "RX"
	Description string     `json:"description"`  // Human-readable description
	Fields      []PDOField `json:"fields"`       // List of fields in this PDO
}

// ParsePDOData parses raw CAN data bytes according to the PDO mapping
func (m *PDOMapping) ParsePDOData(data []byte) map[string]any {
	result := make(map[string]any)

	for _, field := range m.Fields {
		// Check if we have enough data
		if field.ByteOffset+field.ByteLength > len(data) {
			continue
		}

		// Extract bytes for this field
		fieldData := data[field.ByteOffset : field.ByteOffset+field.ByteLength]

		// Parse based on type (assuming little-endian, standard for CANopen)
		var value any
		switch field.Type {
		case FieldTypeInt8:
			value = int8(fieldData[0])
		case FieldTypeUint8:
			value = uint8(fieldData[0])
		case FieldTypeInt16:
			value = int16(binary.LittleEndian.Uint16(fieldData))
		case FieldTypeUint16:
			value = binary.LittleEndian.Uint16(fieldData)
		case FieldTypeInt32:
			value = int32(binary.LittleEndian.Uint32(fieldData))
		case FieldTypeUint32:
			value = binary.LittleEndian.Uint32(fieldData)
		}

		result[field.Name] = value
	}

	return result
}

// PDOMessageType represents a CANopen PDO message type
type PDOMessageType struct {
	PDONumber int
	Direction string // "TX" or "RX"
}

// GetPDOMessageType extracts PDO type from CAN ID
func GetPDOMessageType(canID uint32) *PDOMessageType {
	switch {
	case canID >= 0x180 && canID <= 0x1FF:
		return &PDOMessageType{PDONumber: 1, Direction: "TX"}
	case canID >= 0x200 && canID <= 0x27F:
		return &PDOMessageType{PDONumber: 1, Direction: "RX"}
	case canID >= 0x280 && canID <= 0x2FF:
		return &PDOMessageType{PDONumber: 2, Direction: "TX"}
	case canID >= 0x300 && canID <= 0x37F:
		return &PDOMessageType{PDONumber: 2, Direction: "RX"}
	case canID >= 0x380 && canID <= 0x3FF:
		return &PDOMessageType{PDONumber: 3, Direction: "TX"}
	case canID >= 0x400 && canID <= 0x47F:
		return &PDOMessageType{PDONumber: 3, Direction: "RX"}
	case canID >= 0x480 && canID <= 0x4FF:
		return &PDOMessageType{PDONumber: 4, Direction: "TX"}
	case canID >= 0x500 && canID <= 0x57F:
		return &PDOMessageType{PDONumber: 4, Direction: "RX"}
	}
	return nil
}

