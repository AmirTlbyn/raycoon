package xray

import (
	"encoding/json"
	"fmt"

	"raycoon/internal/core/types"
	"raycoon/internal/storage/models"
)

// XrayConfig represents the root Xray configuration
type XrayConfig struct {
	Log       *LogConfig                `json:"log,omitempty"`
	Stats     *StatsConfig              `json:"stats,omitempty"`
	API       *APIConfig                `json:"api,omitempty"`
	Policy    *PolicyConfig             `json:"policy,omitempty"`
	Inbounds  []InboundConfig           `json:"inbounds"`
	Outbounds []OutboundConfig          `json:"outbounds"`
	Routing   *RoutingConfig            `json:"routing,omitempty"`
	DNS       *DNSConfig                `json:"dns,omitempty"`
}

// StatsConfig enables xray statistics
type StatsConfig struct{}

// APIConfig configures xray gRPC API
type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

// PolicyConfig sets system-level policies
type PolicyConfig struct {
	System *SystemPolicy `json:"system,omitempty"`
}

// SystemPolicy controls system-level stats collection
type SystemPolicy struct {
	StatsInboundUplink    bool `json:"statsInboundUplink"`
	StatsInboundDownlink  bool `json:"statsInboundDownlink"`
	StatsOutboundUplink   bool `json:"statsOutboundUplink"`
	StatsOutboundDownlink bool `json:"statsOutboundDownlink"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	LogLevel string `json:"loglevel"`
}

// InboundConfig represents an inbound configuration
type InboundConfig struct {
	Tag      string                 `json:"tag"`
	Port     int                    `json:"port"`
	Listen   string                 `json:"listen,omitempty"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings,omitempty"`
	Sniffing *SniffingConfig        `json:"sniffing,omitempty"`
}

// SniffingConfig represents traffic sniffing configuration
type SniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
	RouteOnly    bool     `json:"routeOnly,omitempty"`
}

// OutboundConfig represents an outbound configuration
type OutboundConfig struct {
	Tag            string                 `json:"tag"`
	Protocol       string                 `json:"protocol"`
	Settings       map[string]interface{} `json:"settings,omitempty"`
	StreamSettings *StreamSettings        `json:"streamSettings,omitempty"`
	Mux            *MuxConfig             `json:"mux,omitempty"`
}

// MuxConfig represents multiplexing settings
type MuxConfig struct {
	Enabled     bool `json:"enabled"`
	Concurrency int  `json:"concurrency"`
}

// StreamSettings represents stream settings (transport + TLS)
type StreamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security,omitempty"`
	TLSSettings     *TLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *RealitySettings `json:"realitySettings,omitempty"`
	WSSettings      *WSSettings      `json:"wsSettings,omitempty"`
	GRPCSettings    *GRPCSettings    `json:"grpcSettings,omitempty"`
	HTTPSettings    *HTTPSettings    `json:"httpSettings,omitempty"`
	QUICSettings    *QUICSettings    `json:"quicSettings,omitempty"`
	SockoptSettings *SockoptSettings `json:"sockopt,omitempty"`
}

// TLSSettings represents TLS settings
type TLSSettings struct {
	ServerName    string   `json:"serverName,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
	ALPN          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
}

// RealitySettings represents xray Reality protocol settings
type RealitySettings struct {
	ServerName  string `json:"serverName,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortID     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

// SockoptSettings represents socket options (for fragment support)
type SockoptSettings struct {
	DialerProxy     string `json:"dialerProxy,omitempty"`
	TCPKeepAlive    int    `json:"tcpKeepAliveInterval,omitempty"`
	TFO             bool   `json:"tcpFastOpen,omitempty"`
	Fragment        *FragmentSettings `json:"fragment,omitempty"`
}

// FragmentSettings represents TLS fragment settings for anti-censorship
type FragmentSettings struct {
	Packets  string `json:"packets,omitempty"`  // "tlshello", "1-3"
	Length   string `json:"length,omitempty"`   // "100-200"
	Interval string `json:"interval,omitempty"` // "10-20"
}

// WSSettings represents WebSocket settings
type WSSettings struct {
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// GRPCSettings represents gRPC settings
type GRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

// HTTPSettings represents HTTP settings
type HTTPSettings struct {
	Path string   `json:"path,omitempty"`
	Host []string `json:"host,omitempty"`
}

// QUICSettings represents QUIC settings
type QUICSettings struct {
	Security string                 `json:"security,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Header   map[string]interface{} `json:"header,omitempty"`
}

// RoutingConfig represents routing configuration
type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy,omitempty"`
	Rules          []RoutingRule `json:"rules,omitempty"`
}

// RoutingRule represents a routing rule
type RoutingRule struct {
	Type        string   `json:"type,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	IP          []string `json:"ip,omitempty"`
	Port        string   `json:"port,omitempty"`
	Network     string   `json:"network,omitempty"`
	OutboundTag string   `json:"outboundTag"`
	InboundTag  []string `json:"inboundTag,omitempty"`
}

// DNSConfig represents DNS configuration
type DNSConfig struct {
	Servers []interface{} `json:"servers"`
}

// statsAPIPort is the local port used for xray gRPC stats API.
const statsAPIPort = 10085

// generateXrayConfig generates Xray configuration from CoreConfig
func generateXrayConfig(config *types.CoreConfig) (*XrayConfig, error) {
	logLevel := config.LogLevel
	if logLevel == "" {
		logLevel = "none"
	}

	xrayConfig := &XrayConfig{
		Log: &LogConfig{
			LogLevel: logLevel,
		},
		// Enable stats collection for traffic monitoring.
		Stats: &StatsConfig{},
		API: &APIConfig{
			Tag:      "api",
			Services: []string{"StatsService"},
		},
		Policy: &PolicyConfig{
			System: &SystemPolicy{
				StatsInboundUplink:    true,
				StatsInboundDownlink:  true,
				StatsOutboundUplink:   true,
				StatsOutboundDownlink: true,
			},
		},
	}

	// API inbound (dokodemo-door for gRPC stats queries).
	xrayConfig.Inbounds = append(xrayConfig.Inbounds, InboundConfig{
		Tag:      "api-in",
		Port:     statsAPIPort,
		Listen:   "127.0.0.1",
		Protocol: "dokodemo-door",
		Settings: map[string]interface{}{
			"address": "127.0.0.1",
		},
	})

	// Generate inbounds based on VPN mode.
	switch config.VPNMode {
	case types.VPNModeProxy, types.VPNModeTunnel:
		// Both modes use local SOCKS/HTTP inbounds.
		// Tunnel mode additionally sets system proxy (handled by the caller).
		xrayConfig.Inbounds = append(xrayConfig.Inbounds, InboundConfig{
			Tag:      "socks-in",
			Port:     config.SOCKSPort,
			Listen:   "127.0.0.1",
			Protocol: "socks",
			Settings: map[string]interface{}{
				"auth": "noauth",
				"udp":  true,
			},
			Sniffing: &SniffingConfig{
				Enabled:      true,
				DestOverride: []string{"http", "tls"},
				RouteOnly:    true,
			},
		})

		xrayConfig.Inbounds = append(xrayConfig.Inbounds, InboundConfig{
			Tag:      "http-in",
			Port:     config.HTTPPort,
			Listen:   "127.0.0.1",
			Protocol: "http",
			Sniffing: &SniffingConfig{
				Enabled:      true,
				DestOverride: []string{"http", "tls"},
				RouteOnly:    true,
			},
		})

	default:
		return nil, fmt.Errorf("unsupported VPN mode: %s", config.VPNMode)
	}

	// Generate proxy outbound.
	outbound, err := generateOutbound(config.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate outbound: %w", err)
	}
	outbound.Tag = "proxy"

	// Add mux for protocols that benefit from it (not for XTLS/vision flows).
	if shouldEnableMux(config.Config) {
		outbound.Mux = &MuxConfig{
			Enabled:     true,
			Concurrency: 8,
		}
	}

	xrayConfig.Outbounds = append(xrayConfig.Outbounds, *outbound)

	// Direct outbound with UseIPv4 for faster DNS resolution.
	xrayConfig.Outbounds = append(xrayConfig.Outbounds, OutboundConfig{
		Tag:      "direct",
		Protocol: "freedom",
		Settings: map[string]interface{}{
			"domainStrategy": "UseIPv4",
		},
	})

	// Block outbound.
	xrayConfig.Outbounds = append(xrayConfig.Outbounds, OutboundConfig{
		Tag:      "block",
		Protocol: "blackhole",
	})

	// Routing: AsIs for fastest path (no DNS pre-resolution).
	xrayConfig.Routing = &RoutingConfig{
		DomainStrategy: "AsIs",
		Rules:          []RoutingRule{},
	}

	// Route API traffic to the API handler.
	xrayConfig.Routing.Rules = append(xrayConfig.Routing.Rules, RoutingRule{
		Type:        "field",
		InboundTag:  []string{"api-in"},
		OutboundTag: "api",
	})

	// Private IP bypass.
	xrayConfig.Routing.Rules = append(xrayConfig.Routing.Rules, RoutingRule{
		Type:        "field",
		IP:          []string{"geoip:private"},
		OutboundTag: "direct",
	})

	// User-defined routing rules.
	for _, rule := range config.RoutingRules {
		routingRule := RoutingRule{
			Type:        "field",
			OutboundTag: rule.Outbound,
		}
		switch rule.Type {
		case "domain":
			routingRule.Domain = []string{rule.Pattern}
		case "ip":
			routingRule.IP = []string{rule.Pattern}
		}
		xrayConfig.Routing.Rules = append(xrayConfig.Routing.Rules, routingRule)
	}

	// Default: route everything through proxy.
	xrayConfig.Routing.Rules = append(xrayConfig.Routing.Rules, RoutingRule{
		Type:        "field",
		Network:     "tcp,udp",
		OutboundTag: "proxy",
	})

	// DNS: use proxy DNS to avoid leaks, with localhost fallback for local names.
	if len(config.DNSServers) > 0 {
		servers := make([]interface{}, 0, len(config.DNSServers)+1)
		for _, s := range config.DNSServers {
			servers = append(servers, s)
		}
		servers = append(servers, map[string]interface{}{
			"address": "localhost",
			"domains": []string{"geosite:private"},
		})
		xrayConfig.DNS = &DNSConfig{
			Servers: servers,
		}
	}

	return xrayConfig, nil
}

// shouldEnableMux returns true for protocols/transports that benefit from multiplexing.
// Mux must NOT be used with XTLS (vision flow) or QUIC as it breaks them.
func shouldEnableMux(config *models.Config) bool {
	// Never mux QUIC — it already multiplexes.
	if config.Network == "quic" {
		return false
	}

	// VLESS with XTLS flow must not use mux.
	if config.Protocol == "vless" {
		var auth models.AuthConfigVLESS
		if err := json.Unmarshal(config.AuthConfig, &auth); err == nil {
			if auth.Flow != "" {
				return false
			}
		}
	}

	// Enable mux for TCP-based transports.
	switch config.Network {
	case "tcp", "ws", "grpc", "http", "h2", "":
		return true
	}
	return false
}

// generateOutbound generates an outbound configuration from a proxy config
func generateOutbound(config *models.Config) (*OutboundConfig, error) {
	outbound := &OutboundConfig{
		Protocol: config.Protocol,
		Settings: map[string]interface{}{},
	}

	switch config.Protocol {
	case "vmess":
		if err := generateVMessSettings(outbound, config); err != nil {
			return nil, err
		}
	case "vless":
		if err := generateVLESSSettings(outbound, config); err != nil {
			return nil, err
		}
	case "trojan":
		if err := generateTrojanSettings(outbound, config); err != nil {
			return nil, err
		}
	case "shadowsocks":
		if err := generateShadowsocksSettings(outbound, config); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", config.Protocol)
	}

	streamSettings, err := generateStreamSettings(config)
	if err != nil {
		return nil, err
	}
	outbound.StreamSettings = streamSettings

	return outbound, nil
}

func generateVMessSettings(outbound *OutboundConfig, config *models.Config) error {
	var authConfig models.AuthConfigVMess
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return err
	}

	outbound.Settings["vnext"] = []map[string]interface{}{
		{
			"address": config.Address,
			"port":    config.Port,
			"users": []map[string]interface{}{
				{
					"id":       authConfig.UUID,
					"alterId":  authConfig.AlterID,
					"security": authConfig.Security,
				},
			},
		},
	}
	return nil
}

func generateVLESSSettings(outbound *OutboundConfig, config *models.Config) error {
	var authConfig models.AuthConfigVLESS
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return err
	}

	user := map[string]interface{}{
		"id":         authConfig.UUID,
		"encryption": "none",
	}
	if authConfig.Flow != "" {
		user["flow"] = authConfig.Flow
	}

	outbound.Settings["vnext"] = []map[string]interface{}{
		{
			"address": config.Address,
			"port":    config.Port,
			"users":   []map[string]interface{}{user},
		},
	}
	return nil
}

func generateTrojanSettings(outbound *OutboundConfig, config *models.Config) error {
	var authConfig models.AuthConfigTrojan
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return err
	}

	outbound.Settings["servers"] = []map[string]interface{}{
		{
			"address":  config.Address,
			"port":     config.Port,
			"password": authConfig.Password,
		},
	}
	return nil
}

func generateShadowsocksSettings(outbound *OutboundConfig, config *models.Config) error {
	var authConfig models.AuthConfigShadowsocks
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return err
	}

	outbound.Settings["servers"] = []map[string]interface{}{
		{
			"address":  config.Address,
			"port":     config.Port,
			"method":   authConfig.Method,
			"password": authConfig.Password,
		},
	}
	return nil
}

func generateStreamSettings(config *models.Config) (*StreamSettings, error) {
	streamSettings := &StreamSettings{
		Network: config.Network,
	}

	if config.TLSEnabled {
		var tlsConfig models.TLSConfig
		if len(config.TLSConfig) > 0 {
			if err := json.Unmarshal(config.TLSConfig, &tlsConfig); err != nil {
				return nil, err
			}
		}

		// Determine if this is Reality or standard TLS.
		isReality := len(tlsConfig.PublicKey) > 0 && tlsConfig.PublicKey[0] != ""

		if isReality {
			streamSettings.Security = "reality"

			fingerprint := tlsConfig.Fingerprint
			if fingerprint == "" {
				fingerprint = "chrome" // Default to chrome for best compatibility.
			}

			streamSettings.RealitySettings = &RealitySettings{
				ServerName:  tlsConfig.ServerName,
				Fingerprint: fingerprint,
				PublicKey:   tlsConfig.PublicKey[0],
				ShortID:     tlsConfig.ShortID,
				SpiderX:     tlsConfig.SpiderX,
			}
		} else {
			streamSettings.Security = "tls"

			fingerprint := tlsConfig.Fingerprint
			if fingerprint == "" {
				fingerprint = "chrome" // Default to chrome for better anti-detection.
			}

			streamSettings.TLSSettings = &TLSSettings{
				ServerName:    tlsConfig.ServerName,
				AllowInsecure: tlsConfig.AllowInsecure,
				ALPN:          tlsConfig.ALPN,
				Fingerprint:   fingerprint,
			}
		}
	}

	if len(config.TransportConfig) > 0 {
		var transportConfig models.TransportConfig
		if err := json.Unmarshal(config.TransportConfig, &transportConfig); err != nil {
			return nil, err
		}

		switch config.Network {
		case "ws":
			streamSettings.WSSettings = &WSSettings{
				Path:    transportConfig.WSPath,
				Headers: transportConfig.WSHeaders,
			}
		case "grpc":
			streamSettings.GRPCSettings = &GRPCSettings{
				ServiceName: transportConfig.GRPCServiceName,
				MultiMode:   transportConfig.GRPCMode == "multi",
			}
		case "http", "h2":
			streamSettings.HTTPSettings = &HTTPSettings{
				Path: transportConfig.HTTPPath,
			}
			if len(transportConfig.HTTPHeaders) > 0 {
				if hosts, ok := transportConfig.HTTPHeaders["Host"]; ok {
					streamSettings.HTTPSettings.Host = hosts
				}
			}
		case "quic":
			streamSettings.QUICSettings = &QUICSettings{
				Security: transportConfig.QUICSecurity,
				Key:      transportConfig.QUICKey,
			}
		}
	}

	return streamSettings, nil
}

// GenerateTestConfig generates an XrayConfig for latency testing.
func GenerateTestConfig(config *types.CoreConfig) (interface{}, error) {
	return generateXrayConfig(config)
}
