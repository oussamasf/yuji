package utils

import (
	"fmt"
	"strings"
	"unicode"
)

type Command struct {
	Name string
	Args []string
}

func Parser(input string) (*Command, error) {
	var parts []string
	var current strings.Builder
	var inQuotes bool
	var escapeNext bool

	for _, r := range input {
		if escapeNext {
			current.WriteRune(r)
			escapeNext = false
			continue
		}

		switch {
		case r == '\\':
			escapeNext = true
		case r == '"' || r == '\'':
			if inQuotes {
				inQuotes = false
				parts = append(parts, current.String())
				current.Reset()
			} else {
				inQuotes = true
			}
		case unicode.IsSpace(r):
			if inQuotes {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return &Command{Name: parts[0], Args: parts[1:]}, nil
}
