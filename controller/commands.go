package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	configuration "github.com/oussamasf/yuji/config"
	"github.com/oussamasf/yuji/utils"
)

func parsePingArgs() string {
	return "PONG"
}

func parseTypeArgs(args []configuration.RESPValue) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("ERR INVALID_NUMBER_OF_ARGUMENTS")
	}

	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERR MUST_PROVIDE_KEY")
	}

	return key, nil
}

func ParseIncrArgs(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {
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
func parseGetArgs(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {

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
func parseSetArgs(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {
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

	if len(args) > 4 {
		if strings.ToLower(args[3].Value.(string)) == "px" {
			expiry, err := strconv.Atoi(args[4].Value.(string))
			cache[key] = configuration.ICache{ExpirationMap: fmt.Sprintf("%d", expiry)}
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

func parseConfigArgs(args []configuration.RESPValue, dir string, db string) ([]string, error) {
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

func parseEchoArgs(args []configuration.RESPValue) string {
	var echoStr []string
	for _, arg := range args[1:] {
		echoStr = append(echoStr, arg.Value.(string))
	}
	msg := strings.Join(echoStr, " ")

	return msg
}

func parseAddStreamArgs(args []configuration.RESPValue) (string, string, map[string]string, error) {
	keyValue := make(map[string]string)

	if (len(args)%2 == 0) || (len(args) < 3) {
		return "", "", keyValue, fmt.Errorf("ERR Invalid number of stream command arguments")
	}

	streamKey, ok := args[1].Value.(string)
	if !ok {
		return "", "", keyValue, fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")
	}

	newEntryID, ok := args[2].Value.(string)
	if !ok {
		return "", "", keyValue, fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")
	}

	for i := 3; i < len(args); i += 2 {
		key, ok := args[i].Value.(string)
		if !ok {
			return "", "", keyValue, fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")
		}

		value, ok := args[i+1].Value.(string)
		if !ok {
			return "", "", keyValue, fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")

		}
		keyValue[key] = value
	}

	return streamKey, newEntryID, keyValue, nil
}

func parseReadStreamArgs(args []configuration.RESPValue) ([]string, []string, bool, time.Duration, error) {
	var streamKeywordIndex int
	var blockTime time.Duration
	var blockRequested bool
	streamKeys := []string{}
	ids := []string{}

	for i, arg := range args {
		subcommand, _ := arg.Value.(string)

		if strings.ToLower(subcommand) == "block" {
			blockValueStr, _ := args[i+1].Value.(string)
			blockTimeInt, err := strconv.ParseInt(blockValueStr, 10, 64)
			if err != nil {
				return ids, streamKeys, false, time.Duration(0), fmt.Errorf("ERR Invalid block value")
			}

			blockTime = time.Duration(blockTimeInt) * time.Millisecond
			blockRequested = true
		}

		if strings.ToLower(subcommand) == "streams" {
			streamKeywordIndex = i
			break
		}
	}

	for i := streamKeywordIndex + 1; i < len(args); i++ {
		streamKey, ok := args[i].Value.(string)
		if !ok || utils.IsStreamId(streamKey) {
			break
		}
		streamKeys = append(streamKeys, streamKey)
	}

	for i := streamKeywordIndex + 1 + len(streamKeys); i < len(args); i++ {
		id, ok := args[i].Value.(string)
		if !ok || !utils.IsStreamId(id) {
			return ids, streamKeys, false, time.Duration(0), fmt.Errorf("ERR INVALID_ID_TYPE")
		}
		ids = append(ids, id)
	}

	return ids, streamKeys, blockRequested, blockTime, nil
}

func generateReadStreamEntries(id string, entries []configuration.StreamEntry) []string {
	streamResult := []string{}

	for _, entry := range entries {
		if utils.CompareIDs(entry.ID, id) > 0 {
			values := []string{}
			for key, value := range entry.Values {
				values = append(values, fmt.Sprintf("$%d\r\n%s\r\n", len(key), key), fmt.Sprintf("$%d\r\n%s\r\n", len(value), value))
			}

			entryResp := fmt.Sprintf("*%d\r\n$%d\r\n%s\r\n*%d\r\n%s", 2, len(entry.ID), entry.ID, len(values)/2, strings.Join(values, ""))
			streamResult = append(streamResult, entryResp)
		}
	}
	return streamResult

}

func generateReadStreamResponse(ids []string, streamKeys []string, config *configuration.AppSettings) []string {
	results := []string{}
	for i, streamKey := range streamKeys {
		//? Check if the stream exists
		stream, ok := config.RedisMap[streamKey]
		if !ok {
			continue
		}

		streamResult := generateReadStreamEntries(ids[i], stream.StreamData.Entries)

		//? Wrap the stream key and its entries
		if len(streamResult) > 0 {
			keyResp := fmt.Sprintf("*2\r\n$%d\r\n%s\r\n*%d\r\n%s", len(streamKey), streamKey, len(streamResult), strings.Join(streamResult, ""))
			results = append(results, keyResp)
		}
	}
	return results
}

func parseRangeStreamArgs(args []configuration.RESPValue) (string, string, string, error) {
	if len(args) != 4 {
		return "", "", "", fmt.Errorf("ERR Invalid number of stream command arguments")
	}

	streamKey, ok := args[1].Value.(string)
	if !ok {
		return "", "", "", fmt.Errorf("ERR INVALID_ARGUMENT_TYPE")
	}

	startRangeID, _ := args[2].Value.(string)
	endRangeID, _ := args[3].Value.(string)

	return startRangeID, endRangeID, streamKey, nil
}
