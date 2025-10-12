package netutil

import (
	"net"
	"testing"
)

// getFreePort asks the kernel for a free open port that is then used for testing.
func getFreePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("could not resolve tcp addr: %v", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("could not listen on tcp: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestIsPortAvailable(t *testing.T) {
	// Get a known free port
	freePort := getFreePort(t)

	// Test that a free port is reported as available
	if !IsPortAvailable(freePort) {
		t.Errorf("expected port %d to be available, but it was not", freePort)
	}

	// Test that a used port is reported as unavailable
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("could not resolve tcp addr: %v", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("could not listen on tcp: %v", err)
	}
	defer l.Close()
	usedPort := l.Addr().(*net.TCPAddr).Port

	if IsPortAvailable(usedPort) {
		t.Errorf("expected port %d to be unavailable, but it was available", usedPort)
	}
}

func TestFindRandomPort(t *testing.T) {
	port, err := FindRandomPort()
	if err != nil {
		t.Fatalf("FindRandomPort() returned an error: %v", err)
	}

	if port < 40000 || port > 49999 {
		t.Errorf("expected port to be in range [40000, 49999], but got %d", port)
	}

	if !IsPortAvailable(port) {
		t.Errorf("FindRandomPort() returned port %d, but it is not available", port)
	}
}
