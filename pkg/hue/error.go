package hue

import (
	"fmt"
	"strings"
)

// An ErrorList is like [{"error":{"type":101,"address":"","description":"link button not pressed"}}].
type ErrorList struct {
	Errors []Error `json:"errors"`
}

func (e ErrorList) Unwrap() error {
	if len(e.Errors) == 0 {
		return nil
	}

	return e.Errors[0] // other errors are suppressed
}

func (e ErrorList) Error() string {
	var sb strings.Builder
	for _, err := range e.Errors {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}

	str := sb.String()
	if len(str) > 0 {
		return str[:len(str)-1]
	}

	return ""
}

// An Error represents a single error.
type Error struct {
	Type        int    `json:"type"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

func (e Error) Error() string {
	return fmt.Sprintf("hue-error %d (%s): %s", e.Type, e.Address, e.Description)
}
