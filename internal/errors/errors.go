package errors

import "fmt"

type Error struct {
	Op  string
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("operation %q failed: %v", e.Op, e.Err)
}

func E(op string, err error) error {
	return &Error{Op: op, Err: err}
}
