package netutil

import (
	"net"
)

// FindRandomPort asks the kernel for a free open port that is ready to be used.
// It does this by listening on TCP address "127.0.0.1:0", which tells the
// OS to assign an ephemeral port. It then closes the listener and returns
// the assigned port number. This is a race-condition-free way to find an
// available port.
var FindRandomPort = func() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
