package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"raycoon/internal/storage/models"
	"strconv"
	"strings"
)

// VMessParser implements Parser for VMess protocol
type VMessParser struct{}

// vmessJSON represents the VMess JSON structure
type vmessJSON struct {
	V    string `json:"v"`    // Version
	PS   string `json:"ps"`   // Remark/Name
	Add  string `json:"add"`  // Address
	Port interface{} `json:"port"` // Port (can be string or int)
	ID   string `json:"id"`   // UUID
	AID  interface{} `json:"aid"`  // AlterID (can be string or int)
	Scy  string `json:"scy"`  // Security
	Net  string `json:"net"`  // Network type
	Type string `json:"type"` // Header type
	Host string `json:"host"` // Host header
	Path string `json:"path"` // Path
	TLS  string `json:"tls"`  // TLS
	SNI  string `json:"sni"`  // Server name indication
	ALPN string `json:"alpn"` // ALPN
	FP   string `json:"fp"`   // Fingerprint
}

func (p *VMessParser) Protocol() string {
	return "vmess"
}

func (p *VMessParser) Parse(uri string) (*models.Config, error) {
	// VMess URI format: vmess://base64encodedJSON
	if !strings.HasPrefix(uri, "vmess://") {
		return nil, fmt.Errorf("invalid VMess URI: must start with vmess://")
	}

	// Decode base64
	encoded := strings.TrimPrefix(uri, "vmess://")
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		// Try standard base64
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64: %w", err)
			}
		}
	}

	// Parse JSON
	var v vmessJSON
	if err := json.Unmarshal(decoded, &v); err != nil {
		return nil, fmt.Errorf("failed to parse VMess JSON: %w", err)
	}

	// Convert port to int
	port, err := parsePort(v.Port)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// Convert AlterID to int
	alterID, err := parseAlterID(v.AID)
	if err != nil {
		return nil, fmt.Errorf("invalid alter ID: %w", err)
	}

	// Build config
	config := &models.Config{
		Name:     v.PS,
		Protocol: "vmess",
		Address:  v.Add,
		Port:     port,
		Network:  v.Net,
		URI:      uri,
		Enabled:  true,
	}

	// Default network type
	if config.Network == "" {
		config.Network = "tcp"
	}

	// Auth config
	authConfig := models.AuthConfigVMess{
		UUID:     v.ID,
		AlterID:  alterID,
		Security: v.Scy,
	}
	if authConfig.Security == "" {
		authConfig.Security = "auto"
	}

	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth config: %w", err)
	}
	config.AuthConfig = authJSON

	// Transport config
	transportConfig := models.TransportConfig{}
	switch config.Network {
	case "ws":
		transportConfig.WSPath = v.Path
		if v.Host != "" {
			transportConfig.WSHeaders = map[string]string{"Host": v.Host}
		}
	case "grpc":
		transportConfig.GRPCServiceName = v.Path
		transportConfig.GRPCMode = v.Type
	case "http", "h2":
		transportConfig.HTTPPath = v.Path
		if v.Host != "" {
			transportConfig.HTTPHeaders = map[string][]string{"Host": {v.Host}}
		}
	case "quic":
		transportConfig.QUICKey = v.Path
		transportConfig.QUICSecurity = v.Host
	}

	transportJSON, err := json.Marshal(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transport config: %w", err)
	}
	config.TransportConfig = transportJSON

	// TLS config
	config.TLSEnabled = v.TLS == "tls"
	if config.TLSEnabled {
		tlsConfig := models.TLSConfig{
			ServerName:  v.SNI,
			Fingerprint: v.FP,
		}
		if v.ALPN != "" {
			tlsConfig.ALPN = strings.Split(v.ALPN, ",")
		}

		tlsJSON, err := json.Marshal(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal TLS config: %w", err)
		}
		config.TLSConfig = tlsJSON
	}

	// Generate name if empty
	if config.Name == "" {
		config.Name = fmt.Sprintf("%s:%d", config.Address, config.Port)
	}

	return config, nil
}

func (p *VMessParser) Encode(config *models.Config) (string, error) {
	// Unmarshal auth config
	var authConfig models.AuthConfigVMess
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return "", fmt.Errorf("failed to unmarshal auth config: %w", err)
	}

	// Unmarshal transport config
	var transportConfig models.TransportConfig
	if len(config.TransportConfig) > 0 {
		if err := json.Unmarshal(config.TransportConfig, &transportConfig); err != nil {
			return "", fmt.Errorf("failed to unmarshal transport config: %w", err)
		}
	}

	// Unmarshal TLS config
	var tlsConfig models.TLSConfig
	if config.TLSEnabled && len(config.TLSConfig) > 0 {
		if err := json.Unmarshal(config.TLSConfig, &tlsConfig); err != nil {
			return "", fmt.Errorf("failed to unmarshal TLS config: %w", err)
		}
	}

	// Build VMess JSON
	v := vmessJSON{
		V:    "2",
		PS:   config.Name,
		Add:  config.Address,
		Port: config.Port,
		ID:   authConfig.UUID,
		AID:  authConfig.AlterID,
		Scy:  authConfig.Security,
		Net:  config.Network,
	}

	// Transport settings
	switch config.Network {
	case "ws":
		v.Path = transportConfig.WSPath
		if len(transportConfig.WSHeaders) > 0 {
			v.Host = transportConfig.WSHeaders["Host"]
		}
	case "grpc":
		v.Path = transportConfig.GRPCServiceName
		v.Type = transportConfig.GRPCMode
	case "http", "h2":
		v.Path = transportConfig.HTTPPath
		if len(transportConfig.HTTPHeaders) > 0 && len(transportConfig.HTTPHeaders["Host"]) > 0 {
			v.Host = transportConfig.HTTPHeaders["Host"][0]
		}
	case "quic":
		v.Path = transportConfig.QUICKey
		v.Host = transportConfig.QUICSecurity
	}

	// TLS settings
	if config.TLSEnabled {
		v.TLS = "tls"
		v.SNI = tlsConfig.ServerName
		v.FP = tlsConfig.Fingerprint
		if len(tlsConfig.ALPN) > 0 {
			v.ALPN = strings.Join(tlsConfig.ALPN, ",")
		}
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VMess JSON: %w", err)
	}

	// Encode to base64
	encoded := base64.RawURLEncoding.EncodeToString(jsonBytes)

	return "vmess://" + encoded, nil
}

func (p *VMessParser) Validate(config *models.Config) error {
	if config.Protocol != "vmess" {
		return fmt.Errorf("invalid protocol: expected vmess, got %s", config.Protocol)
	}

	if config.Address == "" {
		return fmt.Errorf("address is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	var authConfig models.AuthConfigVMess
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}

	if authConfig.UUID == "" {
		return fmt.Errorf("UUID is required")
	}

	return nil
}

// Helper functions

func parsePort(port interface{}) (int, error) {
	switch v := port.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		p, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return p, nil
	default:
		return 0, fmt.Errorf("unsupported port type: %T", port)
	}
}

func parseAlterID(aid interface{}) (int, error) {
	if aid == nil {
		return 0, nil
	}

	switch v := aid.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		if v == "" {
			return 0, nil
		}
		a, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return a, nil
	default:
		return 0, fmt.Errorf("unsupported alter ID type: %T", aid)
	}
}

func unescapePath(path string) string {
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return decoded
}
