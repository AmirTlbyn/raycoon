package types

import (
	"time"

	"raycoon/internal/storage/models"
)

// CoreConfig represents configuration for starting a core
type CoreConfig struct {
	Config       *models.Config
	VPNMode      VPNMode
	SOCKSPort    int
	HTTPPort     int
	LogLevel     string
	DNSServers   []string
	RoutingRules []RoutingRule
}

// VPNMode represents the VPN operation mode
type VPNMode string

const (
	VPNModeTunnel VPNMode = "tunnel" // System-wide tunneling
	VPNModeProxy  VPNMode = "proxy"  // SOCKS/HTTP proxy
)

// Status represents core runtime status
type Status struct {
	Running   bool
	PID       int
	StartedAt time.Time
	Uptime    time.Duration
	CoreType  string
}

// Stats represents real-time statistics
type Stats struct {
	UploadSpeed   uint64 // bytes per second
	DownloadSpeed uint64 // bytes per second
	TotalUpload   uint64 // total bytes
	TotalDownload uint64 // total bytes
	ActiveConns   int
}

// RoutingRule represents a routing rule
type RoutingRule struct {
	Type     string // domain, ip, geoip, etc.
	Pattern  string
	Outbound string // proxy, direct, block
}

// CoreType represents the type of proxy core
type CoreType string

const (
	CoreTypeXray    CoreType = "xray"
	CoreTypeSingbox CoreType = "singbox"
)
