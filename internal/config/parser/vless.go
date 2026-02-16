package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"raycoon/internal/storage/models"
	"strconv"
	"strings"
)

// VLESSParser implements Parser for VLESS protocol
type VLESSParser struct{}

func (p *VLESSParser) Protocol() string {
	return "vless"
}

func (p *VLESSParser) Parse(uri string) (*models.Config, error) {
	// VLESS URI format: vless://uuid@address:port?parameters#remark
	if !strings.HasPrefix(uri, "vless://") {
		return nil, fmt.Errorf("invalid VLESS URI: must start with vless://")
	}

	// Parse URL
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	// Extract UUID
	uuid := u.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("UUID is required")
	}

	// Extract address and port
	host := u.Hostname()
	portStr := u.Port()
	if host == "" || portStr == "" {
		return nil, fmt.Errorf("address and port are required")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// Parse query parameters
	query := u.Query()

	// Build config
	config := &models.Config{
		Name:     u.Fragment,
		Protocol: "vless",
		Address:  host,
		Port:     port,
		Network:  query.Get("type"),
		URI:      uri,
		Enabled:  true,
	}

	// Default network type
	if config.Network == "" {
		config.Network = "tcp"
	}

	// Auth config
	authConfig := models.AuthConfigVLESS{
		UUID: uuid,
		Flow: query.Get("flow"),
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
		transportConfig.WSPath = query.Get("path")
		if host := query.Get("host"); host != "" {
			transportConfig.WSHeaders = map[string]string{"Host": host}
		}
	case "grpc":
		transportConfig.GRPCServiceName = query.Get("serviceName")
		transportConfig.GRPCMode = query.Get("mode")
	case "http", "h2":
		transportConfig.HTTPPath = query.Get("path")
		if host := query.Get("host"); host != "" {
			transportConfig.HTTPHeaders = map[string][]string{"Host": {host}}
		}
	case "quic":
		transportConfig.QUICKey = query.Get("key")
		transportConfig.QUICSecurity = query.Get("security")
	}

	transportJSON, err := json.Marshal(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transport config: %w", err)
	}
	config.TransportConfig = transportJSON

	// TLS config
	security := query.Get("security")
	config.TLSEnabled = security == "tls" || security == "reality"

	if config.TLSEnabled {
		tlsConfig := models.TLSConfig{
			ServerName:  query.Get("sni"),
			Fingerprint: query.Get("fp"),
		}

		if alpn := query.Get("alpn"); alpn != "" {
			tlsConfig.ALPN = strings.Split(alpn, ",")
		}

		// Reality-specific parameters
		if security == "reality" {
			if pbk := query.Get("pbk"); pbk != "" {
				tlsConfig.PublicKey = []string{pbk}
			}
			tlsConfig.ShortID = query.Get("sid")
			tlsConfig.SpiderX = query.Get("spx")
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
	} else {
		config.Name, _ = url.QueryUnescape(config.Name)
	}

	return config, nil
}

func (p *VLESSParser) Encode(config *models.Config) (string, error) {
	// Unmarshal auth config
	var authConfig models.AuthConfigVLESS
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

	// Build URL
	u := &url.URL{
		Scheme: "vless",
		User:   url.User(authConfig.UUID),
		Host:   fmt.Sprintf("%s:%d", config.Address, config.Port),
	}

	// Build query parameters
	query := url.Values{}

	if config.Network != "tcp" {
		query.Set("type", config.Network)
	}

	if authConfig.Flow != "" {
		query.Set("flow", authConfig.Flow)
	}

	// Transport settings
	switch config.Network {
	case "ws":
		if transportConfig.WSPath != "" {
			query.Set("path", transportConfig.WSPath)
		}
		if len(transportConfig.WSHeaders) > 0 {
			if host, ok := transportConfig.WSHeaders["Host"]; ok {
				query.Set("host", host)
			}
		}
	case "grpc":
		if transportConfig.GRPCServiceName != "" {
			query.Set("serviceName", transportConfig.GRPCServiceName)
		}
		if transportConfig.GRPCMode != "" {
			query.Set("mode", transportConfig.GRPCMode)
		}
	case "http", "h2":
		if transportConfig.HTTPPath != "" {
			query.Set("path", transportConfig.HTTPPath)
		}
		if len(transportConfig.HTTPHeaders) > 0 && len(transportConfig.HTTPHeaders["Host"]) > 0 {
			query.Set("host", transportConfig.HTTPHeaders["Host"][0])
		}
	case "quic":
		if transportConfig.QUICKey != "" {
			query.Set("key", transportConfig.QUICKey)
		}
		if transportConfig.QUICSecurity != "" {
			query.Set("security", transportConfig.QUICSecurity)
		}
	}

	// TLS settings
	if config.TLSEnabled {
		query.Set("security", "tls")
		if tlsConfig.ServerName != "" {
			query.Set("sni", tlsConfig.ServerName)
		}
		if tlsConfig.Fingerprint != "" {
			query.Set("fp", tlsConfig.Fingerprint)
		}
		if len(tlsConfig.ALPN) > 0 {
			query.Set("alpn", strings.Join(tlsConfig.ALPN, ","))
		}

		// Reality parameters
		if len(tlsConfig.PublicKey) > 0 {
			query.Set("security", "reality")
			query.Set("pbk", tlsConfig.PublicKey[0])
		}
		if tlsConfig.ShortID != "" {
			query.Set("sid", tlsConfig.ShortID)
		}
		if tlsConfig.SpiderX != "" {
			query.Set("spx", tlsConfig.SpiderX)
		}
	}

	u.RawQuery = query.Encode()
	u.Fragment = url.QueryEscape(config.Name)

	return u.String(), nil
}

func (p *VLESSParser) Validate(config *models.Config) error {
	if config.Protocol != "vless" {
		return fmt.Errorf("invalid protocol: expected vless, got %s", config.Protocol)
	}

	if config.Address == "" {
		return fmt.Errorf("address is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	var authConfig models.AuthConfigVLESS
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}

	if authConfig.UUID == "" {
		return fmt.Errorf("UUID is required")
	}

	return nil
}
