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

// ShadowsocksParser implements Parser for Shadowsocks protocol
type ShadowsocksParser struct{}

func (p *ShadowsocksParser) Protocol() string {
	return "shadowsocks"
}

func (p *ShadowsocksParser) Parse(uri string) (*models.Config, error) {
	// SS URI format: ss://base64(method:password)@address:port#remark
	// or: ss://base64(method:password@address:port)#remark
	if !strings.HasPrefix(uri, "ss://") && !strings.HasPrefix(uri, "shadowsocks://") {
		return nil, fmt.Errorf("invalid Shadowsocks URI")
	}

	uri = strings.TrimPrefix(uri, "ss://")
	uri = strings.TrimPrefix(uri, "shadowsocks://")

	// Split fragment (remark)
	parts := strings.SplitN(uri, "#", 2)
	remark := ""
	if len(parts) == 2 {
		remark, _ = url.QueryUnescape(parts[1])
		uri = parts[0]
	}

	// Try to parse as ss://base64(method:password)@address:port
	if strings.Contains(uri, "@") {
		parts := strings.SplitN(uri, "@", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid Shadowsocks URI format")
		}

		// Decode userinfo
		decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(parts[0])
			if err != nil {
				decoded, err = base64.StdEncoding.DecodeString(parts[0])
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64: %w", err)
				}
			}
		}

		// Parse method:password
		userinfo := string(decoded)
		credentials := strings.SplitN(userinfo, ":", 2)
		if len(credentials) != 2 {
			return nil, fmt.Errorf("invalid credentials format")
		}

		method := credentials[0]
		password := credentials[1]

		// Parse address:port
		hostPort := strings.SplitN(parts[1], ":", 2)
		if len(hostPort) != 2 {
			return nil, fmt.Errorf("invalid address:port format")
		}

		address := hostPort[0]
		port, err := strconv.Atoi(hostPort[1])
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}

		// Build config
		config := &models.Config{
			Name:     remark,
			Protocol: "shadowsocks",
			Address:  address,
			Port:     port,
			Network:  "tcp",
			URI:      "ss://" + uri,
			Enabled:  true,
		}

		if config.Name == "" {
			config.Name = fmt.Sprintf("%s:%d", address, port)
		}

		authConfig := models.AuthConfigShadowsocks{
			Method:   method,
			Password: password,
		}

		authJSON, _ := json.Marshal(authConfig)
		config.AuthConfig = authJSON

		return config, nil
	}

	return nil, fmt.Errorf("unsupported Shadowsocks URI format")
}

func (p *ShadowsocksParser) Encode(config *models.Config) (string, error) {
	var authConfig models.AuthConfigShadowsocks
	if err := json.Unmarshal(config.AuthConfig, &authConfig); err != nil {
		return "", err
	}

	userinfo := fmt.Sprintf("%s:%s", authConfig.Method, authConfig.Password)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(userinfo))

	uri := fmt.Sprintf("ss://%s@%s:%d#%s",
		encoded, config.Address, config.Port, url.QueryEscape(config.Name))

	return uri, nil
}

func (p *ShadowsocksParser) Validate(config *models.Config) error {
	if config.Protocol != "shadowsocks" {
		return fmt.Errorf("invalid protocol")
	}
	return nil
}
