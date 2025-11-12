package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		err      error
		expected string
	}{
		{
			name:     "simple error",
			op:       "readFile",
			err:      errors.New("file not found"),
			expected: `operation "readFile" failed: file not found`,
		},
		{
			name:     "operation with spaces",
			op:       "write config",
			err:      errors.New("permission denied"),
			expected: `operation "write config" failed: permission denied`,
		},
		{
			name:     "empty operation",
			op:       "",
			err:      errors.New("unknown error"),
			expected: `operation "" failed: unknown error`,
		},
		{
			name:     "nested error",
			op:       "outer",
			err:      E("inner", errors.New("base error")),
			expected: `operation "outer" failed: operation "inner" failed: base error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Error{
				Op:  tt.op,
				Err: tt.err,
			}

			result := e.Error()
			if result != tt.expected {
				t.Errorf("Error.Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestE(t *testing.T) {
	tests := []struct {
		name        string
		op          string
		err         error
		wantContain string
	}{
		{
			name:        "create error with E",
			op:          "testOp",
			err:         errors.New("test error"),
			wantContain: "testOp",
		},
		{
			name:        "create error with nil inner error",
			op:          "someOp",
			err:         nil,
			wantContain: "someOp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := E(tt.op, tt.err)

			// Check that result is of type *Error
			if _, ok := result.(*Error); !ok {
				t.Errorf("E() returned type %T, want *Error", result)
			}

			// Check that the error message contains the operation
			errMsg := result.Error()
			if !strings.Contains(errMsg, tt.wantContain) {
				t.Errorf("E().Error() = %q, want to contain %q", errMsg, tt.wantContain)
			}
		})
	}
}

func TestError_AsError(t *testing.T) {
	// Test that Error implements the error interface
	var err error = &Error{
		Op:  "test",
		Err: errors.New("inner"),
	}

	if err.Error() == "" {
		t.Error("Error should implement error interface and return non-empty string")
	}
}

func TestError_Chaining(t *testing.T) {
	// Test multiple levels of error wrapping
	baseErr := errors.New("base error")
	level1 := E("level1", baseErr)
	level2 := E("level2", level1)
	level3 := E("level3", level2)

	expected := `operation "level3" failed: operation "level2" failed: operation "level1" failed: base error`
	if level3.Error() != expected {
		t.Errorf("Chained error = %q, want %q", level3.Error(), expected)
	}
}

func TestError_Fields(t *testing.T) {
	// Test that Error fields are accessible
	op := "myOperation"
	innerErr := errors.New("inner error")

	e := E(op, innerErr)

	errPtr, ok := e.(*Error)
	if !ok {
		t.Fatal("E() should return *Error type")
	}

	if errPtr.Op != op {
		t.Errorf("Error.Op = %q, want %q", errPtr.Op, op)
	}

	if errPtr.Err != innerErr {
		t.Errorf("Error.Err = %v, want %v", errPtr.Err, innerErr)
	}
}
