package utils

import (
	"encoding/hex"
	"fmt"
)

// Constants for RDB format
const (
	RDBOpcodeSelectDB = 0xFE // DB selection opcode
	RDBOpcodeEOF      = 0xFF // EOF opcode
	RDBOpcodeResizeDB = 0xFC // Resize DB opcode
	RDBTypeString     = 0x00 // String encoding type
)

// extractString reads a length-prefixed string from the data
func extractKeys(data []byte) ([]string, error) {
	var keys []string
	i := 0

	// Skip header (FE 00 FC 00 00 03)
	i += 6

	for i < len(data) {
		if data[i] == RDBOpcodeEOF { // EOF
			break
		}

		if data[i] != RDBTypeString { // Not a string type
			return nil, fmt.Errorf("unexpected data type at position %d", i)
		}
		i++

		// Read key length
		keyLen := int(data[i])
		i++

		// Extract key
		if i+keyLen > len(data) {
			return nil, fmt.Errorf("key length exceeds data bounds at position %d", i)
		}
		key := string(data[i : i+keyLen])
		keys = append(keys, key)
		i += keyLen

		// Skip value length and value
		valueLen := int(data[i])
		i++
		i += valueLen
	}

	return keys, nil
}

func extractBetweenFeAndFf(data []byte) []byte {
	start := -1
	end := -1

	// Find the positions of 'fe' and 'ff'
	for i := 0; i < len(data); i++ {
		if data[i] == 0xfe && start == -1 {
			start = i
		} else if data[i] == 0xff && start != -1 {
			end = i + 1
			break
		}
	}

	// Extract and return the bytes between 'fe' and 'ff'
	if start != -1 && end != -1 && start < end {
		return data[start:end]
	}

	// Return an empty slice if 'fe' or 'ff' are not found or if the range is invalid
	return []byte{}
}

// LogFileKeys reads an RDB file and extracts all keys as strings.
func LogFileKeys() ([]string, error) {
	// Hardcoded RDB file content as a byte slice

	data := []byte{
		0x52, 0x45, 0x44, 0x49, 0x53, 0x30, 0x30, 0x30, 0x36,
		0xFE, 0x00, 0xFC, 0x00, 0x00, 0x03,
		0x00, 0x03, 0x6B, 0x65, 0x79, 0x05, 0x76, 0x61, 0x6C, 0x75, 0x65,
		0x00, 0x04, 0x6E, 0x61, 0x6D, 0x65, 0x04, 0x4A, 0x6F, 0x68, 0x6E,
		0x00, 0x03, 0x61, 0x67, 0x65, 0x02, 0x32, 0x35,
		0xFF,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	fmt.Println(hex.EncodeToString(extractBetweenFeAndFf(data)))
	keys, err := extractKeys(extractBetweenFeAndFf(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse keys: %w", err)
	}

	return keys, nil
}
