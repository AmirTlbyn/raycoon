package latency

import (
	"context"
	"fmt"
	"net"
	"time"

	"raycoon/internal/storage/models"
)

// Strategy defines how a latency test is performed against a single config.
type Strategy interface {
	// Name returns the strategy identifier ("tcp" or "http").
	Name() string
	// Test performs a latency test and returns the round-trip time in milliseconds.
	Test(ctx context.Context, config *models.Config) (latencyMS int, err error)
}

// TCPStrategy measures latency via a TCP handshake to config.Address:config.Port.
// Fast, low overhead - only verifies network reachability without testing proxy protocol.
type TCPStrategy struct{}

func (s *TCPStrategy) Name() string { return "tcp" }

func (s *TCPStrategy) Test(ctx context.Context, config *models.Config) (int, error) {
	address := fmt.Sprintf("%s:%d", config.Address, config.Port)

	start := time.Now()
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return 0, fmt.Errorf("tcp handshake failed: %w", err)
	}
	elapsed := time.Since(start)
	conn.Close()

	return int(elapsed.Milliseconds()), nil
}

// NewStrategy creates a Strategy by name. Valid names: "tcp", "http".
func NewStrategy(name string) (Strategy, error) {
	switch name {
	case "http", "":
		return NewHTTPStrategy()
	case "tcp":
		return &TCPStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown test strategy: %s (available: tcp, http)", name)
	}
}
