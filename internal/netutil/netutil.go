package netutil

import (
	"fmt"
	"net"
	"time"
)

// FindRandomPort finds an available TCP port in the range 40000-49999.
func FindRandomPort() (int, error) {
	for i := 0; i < 100; i++ { // Try 100 times
		port := 40000 + int(time.Now().UnixNano()%10000)
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("could not find a free port after 100 attempts")
}

// IsPortAvailable checks if a TCP port is available to be listened on.
func IsPortAvailable(port int) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
