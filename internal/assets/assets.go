package assets

import (
	_ "embed"
)

//go:embed launchd/socket_vmnet_wrapper.sh
var SocketVMNetWrapper []byte

//go:embed launchd/io.github.pallotron.pvmlab.socket_vmnet.plist
var SocketVMNetPlist []byte
