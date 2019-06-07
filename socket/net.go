/*-
 * Copyright 2019 Square Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package socket

import (
	"net"
	"strings"

	reuseport "github.com/kavu/go_reuseport"
)

// ParseAddress parses a string representing a TCP address or UNIX socket
// for our backend target. The input can be or the form "HOST:PORT" for
// TCP or "unix:PATH" for a UNIX socket. It also accepts 'launchd' or
// 'systemd' for socket activation with those systems.
func ParseAddress(input string) (network, address, host string, err error) {
	if input == "launchd" || input == "systemd" {
		network = input
		return
	}

	if strings.HasPrefix(input, "unix:") {
		network = "unix"
		address = input[5:]
		return
	}

	host, _, err = net.SplitHostPort(input)
	if err != nil {
		return
	}

	// Make sure target address resolves
	_, err = net.ResolveTCPAddr("tcp", input)
	if err != nil {
		return
	}

	network, address = "tcp", input
	return
}

// Open a listening socket with the given network and address.
// Supports 'unix', 'tcp', 'launchd' and 'systemd' as the network.
//
// For 'tcp' sockets, the address must be a host and a port. The
// opened socket will be bound with SO_REUSEPORT.
//
// For 'unix' sockets, the address must be a path. The socket file
// will be set to unlink on close automatically.
//
// For 'launchd' and 'systemd' sockets, the address must be empty.
// The actual socket will come from launchd or systemd, which must
// be configured for socket activation.
func Open(network, address string) (net.Listener, error) {
	switch network {
	case "launchd":
		return launchdSocket()
	case "systemd":
		return systemdSocket()
	case "unix":
		listener, err := net.Listen(network, address)
		listener.(*net.UnixListener).SetUnlinkOnClose(true)
		return listener, err
	default:
		return reuseport.NewReusablePortListener(network, address)
	}
}

// ParseAndOpen combines the functionality of the ParseAddress and Open methods.
func ParseAndOpen(address string) (net.Listener, error) {
	net, addr, _, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	return Open(net, addr)
}
