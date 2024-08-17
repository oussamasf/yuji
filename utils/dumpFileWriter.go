package utils

import (
	"bytes"
	"encoding/binary"
	"hash/crc64"
	"os"
	"path/filepath"
	"time"
)

const (
	redisMagic          = "REDIS"
	rdbVersion          = "0009" // Example version 9
	typeString          = 0x00   // Type for Redis string
	dbSubsection        = 0xFE   // Start of a database subsection
	hashTableSizeMarker = 0xFB   // Indicates hash table size information
	expireMillis        = 0xFC   // Indicates an expire timestamp in milliseconds
	expireSeconds       = 0xFD   // Indicates an expire timestamp in seconds
)

// SaveRDBFile saves each key-value pair in the map with the specified format.
// Parameters:
// - filePath: the path where the RDB file should be saved.
// - dbIndex: the logical Redis database index (usually an integer like 0).
// - data: the hashmap containing the keys and their associated values.
// - expirations: map of keys with expiration times, in Unix milliseconds or seconds.
func SaveRDBFile(dbIndex int, config *Config) error {
	var buf bytes.Buffer

	//? Write the Redis RDB header
	buf.WriteString(redisMagic)
	buf.WriteString(rdbVersion)

	//? Write the database subsection
	buf.WriteByte(dbSubsection)
	buf.WriteByte(byte(dbIndex))

	//? Write hash table size information
	buf.WriteByte(hashTableSizeMarker)
	writeLength(&buf, len(config.RedisMap))      // Hash table size (number of key-value pairs)
	writeLength(&buf, len(config.ExpirationMap)) // Expiration hash table size

	//? Write each key-value pair
	for key, value := range config.RedisMap {
		//? Check if the key has an expiration
		if expiration, exists := config.ExpirationMap[key]; exists {
			//? Write either milliseconds or seconds expiration
			if isMillis(expiration) {
				buf.WriteByte(expireMillis)
				binary.Write(&buf, binary.LittleEndian, uint64(expiration))
			} else {
				buf.WriteByte(expireSeconds)
				binary.Write(&buf, binary.LittleEndian, uint32(expiration))
			}
		}

		//? Write the key-value pair
		buf.WriteByte(typeString)
		writeLengthPrefixedString(&buf, key)
		writeLengthPrefixedString(&buf, value)
	}

	//? Write the RDB footer (checksum and EOF marker)
	checksum := crc64.Checksum(buf.Bytes(), crc64.MakeTable(crc64.ISO))
	binary.Write(&buf, binary.LittleEndian, checksum)
	buf.WriteByte(0xFF) // EOF marker

	//? Save the RDB file

	//? Check if the directory exists
	if _, err := os.Stat(config.Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(config.Dir, 0755); err != nil {
			return err
		}
	}

	fullPath := filepath.Join(config.Dir, config.DBFileName)
	return os.WriteFile(fullPath, buf.Bytes(), 0644)
}

// ? Helper function to determine if expiration is in milliseconds
func isMillis(expiration int64) bool {
	return expiration > time.Now().Unix()*1000
}

// ? Helper function to write a length-prefixed string
func writeLengthPrefixedString(buf *bytes.Buffer, str string) {
	writeLength(buf, len(str))
	buf.WriteString(str)
}

// ? Helper function to write lengths based on the RDB encoding scheme
func writeLength(buf *bytes.Buffer, length int) {
	if length < 0x80 {
		buf.WriteByte(byte(length))
	} else if length < 0x4000 {
		buf.WriteByte(byte((length >> 8) | 0x80))
		buf.WriteByte(byte(length & 0xFF))
	} else {
		buf.WriteByte(0xC0)
		binary.Write(buf, binary.BigEndian, uint32(length))
	}
}
