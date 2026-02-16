package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"raycoon/internal/storage/models"
	"strconv"
	"strings"
)

// TrojanParser implements Parser for Trojan protocol
type TrojanParser struct{}

func (p *TrojanParser) Protocol() string {
	return "trojan"
}

func (p *TrojanParser) Parse(uri string) (*models.Config, error) {
	// Trojan URI format: trojan://password@address:port?parameters#remark
	if !strings.HasPrefix(uri, "trojan://") {
		return nil, fmt.Errorf("invalid Trojan URI: must start with trojan://")
	}

	// Parse URL
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	// Extract password
	password := u.User.Username()
	if password == "" {
		return nil, fmt.Errorf("password is required")
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
		Protocol: "trojan",
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
	authConfig := models.AuthConfigTrojan{
		Password: password,
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
	}

	transportJSON, err := json.Marshal(transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transport config: %w", err)
	}
	config.TransportConfig = transportJSON

	// TLS config (Trojan typically uses TLS)
	security := query.Get("security")
	config.TLSEnabled = security != "none" // TLS enabled by default for Trojan

	if config.TLSEnabled {
		tlsConfig := models.TLSConfig{
			ServerName: query.Get("sni"),
		}

		if tlsConfig.ServerName == "" {
			tlsConfig.ServerName = host
		}

		if alpn := query.Get("alpn"); alpn != "" {
			tlsConfig.ALPN = strings.Split(alpn, ",")
		}

		if fp := query.Get("fp"); fp != "" {
			tlsConfig.Fingerprint = fp
		}

		// Allow insecure
		if allowInsecure := query.Get("allowInsecure"); allowInsecure == "1" || allowInsecure == "true" {
			tlsConfig.AllowInsecure = true
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

func (p *TrojanParser) Encode(config *models.Config) (string, error) {
	// Unmarshal auth config
	var authConfig models.AuthConfigTrojan
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
		Scheme: "trojan",
		User:   url.User(authConfig.Password),
		Host:   fmt.Sprintf("%s:%d", config.Address, config.Port),
	}

	// Build query parameters
	query := url.Values{}

	if config.Network != "tcp" {
		query.Set("type", config.Network)
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
	}

	// TLS settings
	if config.TLSEnabled {
		if tlsConfig.ServerName != "" {
			query.Set("sni", tlsConfig.ServerName)
		}
		if tlsConfig.Fingerprint != "" {
			query.Set("fp", tlsConfig.Fingerprint)
		}
		if len(tlsConfig.ALPN) > 0 {
			query.Set("alpn", strings.Join(tlsConfig.ALPN, ","))
		}
		if tlsConfig.AllowInsecure {
			query.Set("allowInsecure", "1")
		}
	} else {
		query.Set("security", "none")
	}

	u.RawQuery = query.Encode()
	u.Fragment = url.QueryEscape(config.Name)

	return u.String(), nil
}

func (p *TrojanParser) Validate(config *models.Config) error {
	if config.Protocol != "trojan" {
		return fmt.Errorf("invalid protocol: expected trojan, got %s", config.Protocol)
	}

	if config.Address == "" {
		return fmt.Errorf("address is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	var authConfig models.AuthConfigTrojan
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}

	if authConfig.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}
