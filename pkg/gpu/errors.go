package gpu

import (
	"errors"
	"fmt"
)

var ErrUnavailable = errors.New("gpu loader unavailable")

type OpError struct {
	Vendor  string
	Op      string
	Command string
	Err     error
}

func (e *OpError) Error() string {
	message := e.Vendor
	if e.Op != "" {
		message += " " + e.Op
	}
	if e.Command != "" {
		message += " " + e.Command
	}
	if message == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", message, e.Err)
}

func (e *OpError) Unwrap() error {
	return e.Err
}
