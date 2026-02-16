package models

import (
	"encoding/json"
	"time"
)

// Config represents a proxy configuration
type Config struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Protocol string `json:"protocol"` // vmess, vless, trojan, ss, hysteria2, tuic, wireguard
	GroupID int64  `json:"group_id"`

	// Connection details
	Address string `json:"address"`
	Port    int    `json:"port"`

	// Auth details (protocol-specific, stored as JSON)
	AuthConfig json.RawMessage `json:"auth_config"`

	// Transport details
	Network         string          `json:"network"` // tcp, ws, grpc, http, quic
	TransportConfig json.RawMessage `json:"transport_config,omitempty"`

	// TLS details
	TLSEnabled bool            `json:"tls_enabled"`
	TLSConfig  json.RawMessage `json:"tls_config,omitempty"`

	// Original URI
	URI string `json:"uri,omitempty"`

	// Source tracking
	FromSubscription bool `json:"from_subscription"` // true if from group's subscription, false if manually added

	// Metadata
	Enabled bool     `json:"enabled"`
	Tags    []string `json:"tags,omitempty"`
	Notes   string   `json:"notes,omitempty"`

	// Stats
	LastUsed *time.Time `json:"last_used,omitempty"`
	UseCount int        `json:"use_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuthConfigVMess represents VMess authentication configuration
type AuthConfigVMess struct {
	UUID       string `json:"uuid"`
	AlterID    int    `json:"alter_id"`
	Security   string `json:"security"`    // auto, aes-128-gcm, chacha20-poly1305, none
}

// AuthConfigVLESS represents VLESS authentication configuration
type AuthConfigVLESS struct {
	UUID string `json:"uuid"`
	Flow string `json:"flow,omitempty"` // xtls-rprx-vision, etc.
}

// AuthConfigTrojan represents Trojan authentication configuration
type AuthConfigTrojan struct {
	Password string `json:"password"`
}

// AuthConfigShadowsocks represents Shadowsocks authentication configuration
type AuthConfigShadowsocks struct {
	Method   string `json:"method"`   // encryption method
	Password string `json:"password"`
}

// AuthConfigHysteria2 represents Hysteria2 authentication configuration
type AuthConfigHysteria2 struct {
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// AuthConfigTUIC represents TUIC authentication configuration
type AuthConfigTUIC struct {
	UUID     string `json:"uuid"`
	Password string `json:"password,omitempty"`
}

// AuthConfigWireGuard represents WireGuard authentication configuration
type AuthConfigWireGuard struct {
	PrivateKey string   `json:"private_key"`
	PublicKey  string   `json:"public_key"`
	PeerPublicKey string `json:"peer_public_key"`
	PreSharedKey  string `json:"pre_shared_key,omitempty"`
}

// TransportConfig represents generic transport configuration
type TransportConfig struct {
	// WebSocket
	WSPath    string            `json:"ws_path,omitempty"`
	WSHeaders map[string]string `json:"ws_headers,omitempty"`

	// gRPC
	GRPCServiceName string `json:"grpc_service_name,omitempty"`
	GRPCMode        string `json:"grpc_mode,omitempty"` // gun, multi

	// HTTP
	HTTPPath    string              `json:"http_path,omitempty"`
	HTTPHeaders map[string][]string `json:"http_headers,omitempty"`

	// QUIC
	QUICKey      string `json:"quic_key,omitempty"`
	QUICSecurity string `json:"quic_security,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	ServerName       string   `json:"server_name,omitempty"`
	ALPN             []string `json:"alpn,omitempty"`
	AllowInsecure    bool     `json:"allow_insecure"`
	Fingerprint      string   `json:"fingerprint,omitempty"`
	PublicKey        []string `json:"public_key,omitempty"`
	ShortID          string   `json:"short_id,omitempty"`
	SpiderX          string   `json:"spider_x,omitempty"`
}
