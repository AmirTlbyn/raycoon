package models

import "time"

// ActiveConnection represents the currently active proxy connection
type ActiveConnection struct {
	ID        int64     `json:"id"` // Always 1 (singleton)
	ConfigID  int64     `json:"config_id"`
	CoreType  string    `json:"core_type"`  // xray, singbox
	VPNMode   string    `json:"vpn_mode"`   // tun, proxy
	StartedAt time.Time `json:"started_at"`
}
