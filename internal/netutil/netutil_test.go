package netutil

import (
	"testing"
)

func TestFindRandomPort(t *testing.T) {
	port, err := FindRandomPort()
	if err != nil {
		t.Fatalf("FindRandomPort() returned an error: %v", err)
	}

	if port <= 0 || port > 65535 {
		t.Errorf("FindRandomPort() returned an invalid port number: %d", port)
	}

	// Try to get another port to ensure it's not always the same
	port2, err := FindRandomPort()
	if err != nil {
		t.Fatalf("FindRandomPort() returned an error on second call: %v", err)
	}

	if port2 <= 0 || port2 > 65535 {
		t.Errorf("FindRandomPort() returned an invalid port number on second call: %d", port2)
	}

	// It's highly unlikely, but not impossible, for the OS to return the same
	// ephemeral port twice in a row. This is not a robust check for uniqueness,
	// but it's a basic sanity check.
	if port == port2 {
		t.Logf("Warning: FindRandomPort() returned the same port twice in a row: %d. This is unlikely but possible.", port)
	}

	t.Logf("Successfully found two random ports: %d and %d", port, port2)
}