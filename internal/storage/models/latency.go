package models

import "time"

// LatencyTest represents a latency test result
type LatencyTest struct {
	ID           int64      `json:"id"`
	ConfigID     int64      `json:"config_id"`
	LatencyMS    *int       `json:"latency_ms,omitempty"` // NULL if failed
	Success      bool       `json:"success"`
	ErrorMessage string     `json:"error_message,omitempty"`
	TestStrategy string     `json:"test_strategy"` // tcp, http, icmp
	TestedAt     time.Time  `json:"tested_at"`
}
