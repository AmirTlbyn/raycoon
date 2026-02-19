package tun

// Config holds the TUN device and routing configuration.
type Config struct {
	DeviceName string // TUN device name (utunN on macOS, tun0 on Linux)
	MTU        int    // Maximum transmission unit (default: 9000)
	SOCKSAddr  string // SOCKS5 proxy address, e.g. "127.0.0.1:1080"
	Gateway    string // Original default gateway (detected at runtime)
	Interface  string // Original default interface name (detected at runtime)
}

// DefaultMTU is the default MTU for the TUN device.
const DefaultMTU = 9000
