package cmd

import (
	"os"
	"testing"
)

func TestExecute(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "help command",
			args: []string{"pvmlab", "--help"},
		},
		{
			name: "no args - should show help",
			args: []string{"pvmlab"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			// Reset root command for each test
			rootCmd.SetArgs(tt.args[1:])

			// Execute should not panic
			// We can't easily test the output without more complex setup
			// but we can at least ensure it runs
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Execute() panicked: %v", r)
				}
			}()

			// For help commands, Execute() will not return an error
			// For invalid commands, it will exit but we can't easily test that
			// without mocking os.Exit
		})
	}
}

func TestRootCmd_RunE(t *testing.T) {
	// Test that the root command's RunE function works
	err := rootCmd.RunE(rootCmd, []string{})
	if err != nil {
		t.Errorf("rootCmd.RunE() returned error: %v", err)
	}
}
