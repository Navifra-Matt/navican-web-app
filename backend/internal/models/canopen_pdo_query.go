package models

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePDOFieldsFromQuery parses PDO field definitions from URL query parameter
// Format: "field_name:type:offset:length,field_name2:type:offset:length"
// Example: "statusword:uint16:0:2,mode_of_operation:int8:2:1"
func ParsePDOFieldsFromQuery(queryValue string) ([]PDOField, error) {
	if queryValue == "" {
		return nil, nil
	}

	fields := []PDOField{}
	fieldDefs := strings.Split(queryValue, ",")

	for _, fieldDef := range fieldDefs {
		parts := strings.Split(strings.TrimSpace(fieldDef), ":")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid field definition '%s', expected format: name:type:offset:length", fieldDef)
		}

		name := strings.TrimSpace(parts[0])
		typeStr := strings.TrimSpace(parts[1])
		offsetStr := strings.TrimSpace(parts[2])
		lengthStr := strings.TrimSpace(parts[3])

		// Validate type
		fieldType := PDOFieldType(typeStr)
		validTypes := []PDOFieldType{FieldTypeInt8, FieldTypeUint8, FieldTypeInt16, FieldTypeUint16, FieldTypeInt32, FieldTypeUint32}
		isValidType := false
		for _, vt := range validTypes {
			if fieldType == vt {
				isValidType = true
				break
			}
		}
		if !isValidType {
			return nil, fmt.Errorf("invalid field type '%s', must be one of: int8, uint8, int16, uint16, int32, uint32", typeStr)
		}

		// Parse offset
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 || offset > 7 {
			return nil, fmt.Errorf("invalid byte offset '%s', must be 0-7", offsetStr)
		}

		// Parse length
		length, err := strconv.Atoi(lengthStr)
		if err != nil || length < 1 || length > 8 {
			return nil, fmt.Errorf("invalid byte length '%s', must be 1-8", lengthStr)
		}

		// Validate offset + length doesn't exceed 8 bytes
		if offset+length > 8 {
			return nil, fmt.Errorf("field '%s' exceeds 8-byte CAN data limit (offset %d + length %d)", name, offset, length)
		}

		fields = append(fields, PDOField{
			Name:       name,
			Type:       fieldType,
			ByteOffset: offset,
			ByteLength: length,
		})
	}

	return fields, nil
}

// CreatePDOMappingFromQuery creates a PDO mapping from query parameters
func CreatePDOMappingFromQuery(pdoNumber int, direction string, fields []PDOField) *PDOMapping {
	return &PDOMapping{
		PDONumber:   pdoNumber,
		Direction:   direction,
		Description: fmt.Sprintf("Custom PDO%d_%s from query", pdoNumber, direction),
		Fields:      fields,
	}
}
