package controller

import (
	"fmt"
	"strings"

	configuration "github.com/oussamasf/yuji/config"
	"github.com/oussamasf/yuji/utils"
)

func validateCommand(data []byte) (string, []configuration.RESPValue, error) {

	formattedInput := strings.ReplaceAll(string(data), "\\r\\n", "\r\n")

	commands, err := utils.Parser(formattedInput)

	if err != nil {
		return "", []configuration.RESPValue{}, fmt.Errorf("ERR Invalid command")
	}

	if commands.Type != '*' {
		return "", []configuration.RESPValue{}, fmt.Errorf("ERR Expected array for command")
	}

	args, ok := commands.Value.([]configuration.RESPValue)
	if !ok {
		return "", []configuration.RESPValue{}, fmt.Errorf("ERR Invalid command format")
	}

	if len(args) < 1 {
		return "", []configuration.RESPValue{}, fmt.Errorf("ERR No command given")
	}

	cmdName, ok := args[0].Value.(string)
	if !ok {
		return "", []configuration.RESPValue{}, fmt.Errorf("ERR Invalid command name")

	}
	return strings.ToLower(cmdName), args, nil
}
