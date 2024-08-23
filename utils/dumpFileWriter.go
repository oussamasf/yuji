package utils

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	configuration "github.com/oussamasf/yuji/config"
)

func SaveRDBFile(config *configuration.AppSettings) error {
	if _, err := os.Stat(config.Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(config.Dir, 0755); err != nil {
			return err
		}
	}

	fullPath := filepath.Join(config.Dir, config.DBFileName)

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	//? Write database subsection start
	file.Write([]byte{0xFE, 0x00})

	//? Write hash table sizes
	file.Write([]byte{0xFB})
	writeSize(file, uint64(len(config.RedisMap)))
	writeSize(file, uint64(len(config.RedisMap)))

	//? Write key-value pairs
	for key, value := range config.RedisMap {
		//? Write string type flag
		file.Write([]byte{0x00})

		writeString(file, key)

		writeString(file, value.Data)

		//? Write expire if exists
		expireMs, err := strconv.ParseInt(config.RedisMap[key].ExpirationMap, 10, 64)

		if err == nil {
			expirationTime := time.Unix(0, expireMs*int64(time.Millisecond))
			now := time.Now()
			if expirationTime.Sub(now) > 1000*time.Hour {
				file.Write([]byte{0xFC})
				binary.Write(file, binary.LittleEndian, uint64(expirationTime.UnixNano()/1e6))
			} else {
				file.Write([]byte{0xFD})
				binary.Write(file, binary.LittleEndian, uint32(expirationTime.Unix()))
			}
		}

	}

	return nil
}

func writeSize(file *os.File, size uint64) {
	binary.Write(file, binary.BigEndian, size)
}

func writeString(file *os.File, s string) {
	writeSize(file, uint64(len(s)))
	file.Write([]byte(s))
}
