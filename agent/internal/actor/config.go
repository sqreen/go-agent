package actor

import "net"

// Maximum number of actions that can be stored.
const maxStoreActions = 1024 * 1024

// Number of bits in IP addresses.
const (
	ipv4Bits = net.IPv4len * 8
	ipv6Bits = net.IPv6len * 8
)
