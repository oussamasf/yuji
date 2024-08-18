package utils

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	configuration "github.com/oussamasf/yuji/config"
)

func Parser(input string) (*configuration.RESPValue, error) {
	reader := bufio.NewReader(strings.NewReader(input))
	return parseRESPValue(reader)
}

func parseRESPValue(reader *bufio.Reader) (*configuration.RESPValue, error) {
	dataType, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch dataType {
	case '+':
		return parseSimpleString(reader)
	case '-':
		return parseError(reader)
	case ':':
		return parseInteger(reader)
	case '$':
		return parseBulkString(reader)
	case '*':
		return parseArray(reader)
	default:
		return nil, fmt.Errorf("unknown data type: %c", dataType)
	}
}

func parseSimpleString(reader *bufio.Reader) (*configuration.RESPValue, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return &configuration.RESPValue{Type: '+', Value: strings.TrimRight(line, "\r\n")}, nil
}

func parseError(reader *bufio.Reader) (*configuration.RESPValue, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return &configuration.RESPValue{Type: '-', Value: strings.TrimRight(line, "\r\n")}, nil
}

func parseInteger(reader *bufio.Reader) (*configuration.RESPValue, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseInt(strings.TrimRight(line, "\r\n"), 10, 64)
	if err != nil {
		return nil, err
	}
	return &configuration.RESPValue{Type: ':', Value: value}, nil
}

func parseBulkString(reader *bufio.Reader) (*configuration.RESPValue, error) {
	lenStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	length, err := strconv.Atoi(strings.TrimRight(lenStr, "\r\n"))
	if err != nil {
		return nil, err
	}
	if length == -1 {
		return &configuration.RESPValue{Type: '$', Value: nil}, nil
	}
	data := make([]byte, length+2) // +2 for \r\n
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return nil, err
	}
	return &configuration.RESPValue{Type: '$', Value: string(data[:length])}, nil
}

func parseArray(reader *bufio.Reader) (*configuration.RESPValue, error) {
	lenStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	length, err := strconv.Atoi(strings.TrimRight(lenStr, "\r\n"))
	if err != nil {
		return nil, err
	}
	if length == -1 {
		return &configuration.RESPValue{Type: '*', Value: nil}, nil
	}
	array := make([]configuration.RESPValue, length)
	for i := 0; i < length; i++ {
		value, err := parseRESPValue(reader)
		if err != nil {
			return nil, err
		}
		array[i] = *value
	}
	return &configuration.RESPValue{Type: '*', Value: array}, nil
}
