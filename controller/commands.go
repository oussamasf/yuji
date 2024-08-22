package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	configuration "github.com/oussamasf/yuji/config"
)

func handlePingCmd() string {
	return "PONG"
}

func handleTypeCmd(args []configuration.RESPValue) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("ERR INVALID_NUMBER_OF_ARGUMENTS")
	}

	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERR MUST_PROVIDE_KEY")
	}

	return key, nil
}

func handleIncrCmd(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}
	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}
	result, exists := cache[key]
	if !exists {
		cache[key] = configuration.ICache{
			Data: "1",
		}
	} else {
		intValue, err := strconv.Atoi(result.Data)
		if err != nil {
			return "", fmt.Errorf("ERROR: CANNOT_INCR_NOT_INT")
		}
		cache[key] = configuration.ICache{
			Data: strconv.Itoa(intValue + 1),
		}

	}

	return cache[key].Data, nil
}

// ? GET
func handleGetCmd(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {

	if len(args) != 2 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}
	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}
	result := cache[key]
	return result.Data, nil
}

// ? SET
func handleSetCmd(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {
	if len(args) < 3 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}

	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}

	value, ok := args[2].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}

	cache[key] = configuration.ICache{
		Data: value,
		Type: configuration.CacheDataType(1),
	}

	// TODO change this to store also px in db
	if len(args) > 4 {
		if strings.ToLower(args[3].Value.(string)) == "px" {
			expiry, err := strconv.Atoi(args[4].Value.(string))
			if err != nil {
				return "", fmt.Errorf("ERROR: INVALID_PX")
			}
			time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
				delete(cache, key)
			})
		} else {
			return "", fmt.Errorf("ERROR: INVALID_ARGUMENT")
		}
	}

	return "OK", nil
}

func handleConfigCmd(args []configuration.RESPValue, dir string, db string) ([]string, error) {
	subcommand, ok := args[1].Value.(string)
	subcommand = strings.ToLower(subcommand)
	if !ok && subcommand != "get" {
		return []string{}, fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")

	}

	value, ok := args[2].Value.(string)
	value = strings.ToLower(value)

	if !ok {
		return []string{}, fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")
	}

	if value == "dir" {
		return []string{"dir", fmt.Sprint(len(dir)), dir}, nil

	} else if value == "dbfilename" {
		return []string{"dbfilename", fmt.Sprint(len(db)), db}, nil

	} else {
		return []string{}, fmt.Errorf("ERR unsupported CONFIG parameter")

	}
}

func handleEchoCmd(args []configuration.RESPValue) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("ERR INVALID_NUMBER_OF_ARGUMENTS")
	}
	msg, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")
	}
	return msg, nil

}
