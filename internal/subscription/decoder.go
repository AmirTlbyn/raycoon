package subscription

import (
	"encoding/base64"
	"fmt"
	"strings"

	pkgerrors "raycoon/pkg/errors"
)

// Decoder handles decoding subscription content
type Decoder struct{}

// NewDecoder creates a new subscription decoder
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode decodes subscription content and returns a list of URIs
func (d *Decoder) Decode(content []byte) ([]string, error) {
	if len(content) == 0 {
		return nil, pkgerrors.ErrSubscriptionEmpty
	}

	// Try to decode as base64
	decoded, err := d.decodeBase64(content)
	if err != nil {
		// If base64 decode fails, assume it's already plain text
		decoded = string(content)
	}

	// Split by newlines and filter empty lines
	lines := strings.Split(decoded, "\n")
	uris := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line looks like a valid URI
		if d.isValidURI(line) {
			uris = append(uris, line)
		}
	}

	if len(uris) == 0 {
		return nil, pkgerrors.ErrSubscriptionEmpty
	}

	return uris, nil
}

// decodeBase64 attempts to decode base64 content
func (d *Decoder) decodeBase64(content []byte) (string, error) {
	contentStr := string(content)

	// Try different base64 encodings
	decoders := []func(string) ([]byte, error){
		// Standard base64
		func(s string) ([]byte, error) {
			return base64.StdEncoding.DecodeString(s)
		},
		// URL-safe base64
		func(s string) ([]byte, error) {
			return base64.URLEncoding.DecodeString(s)
		},
		// Raw standard base64 (no padding)
		func(s string) ([]byte, error) {
			return base64.RawStdEncoding.DecodeString(s)
		},
		// Raw URL-safe base64 (no padding)
		func(s string) ([]byte, error) {
			return base64.RawURLEncoding.DecodeString(s)
		},
	}

	for _, decoder := range decoders {
		if decoded, err := decoder(contentStr); err == nil {
			return string(decoded), nil
		}
	}

	return "", fmt.Errorf("failed to decode base64")
}

// isValidURI checks if a string looks like a valid proxy URI
func (d *Decoder) isValidURI(uri string) bool {
	// Check for known protocol prefixes
	protocols := []string{
		"vmess://",
		"vless://",
		"trojan://",
		"ss://",
		"shadowsocks://",
		"hysteria://",
		"hysteria2://",
		"hy2://",
		"tuic://",
		"wireguard://",
	}

	for _, protocol := range protocols {
		if strings.HasPrefix(strings.ToLower(uri), protocol) {
			return true
		}
	}

	return false
}

// ExtractMetadata extracts metadata from subscription headers
func (d *Decoder) ExtractMetadata(headers map[string][]string) *SubscriptionMetadata {
	metadata := &SubscriptionMetadata{
		Headers: headers,
	}

	// Extract common metadata
	if val := headers["Subscription-Userinfo"]; len(val) > 0 {
		metadata.UserInfo = val[0]
	}

	if val := headers["Profile-Update-Interval"]; len(val) > 0 {
		metadata.UpdateInterval = val[0]
	}

	if val := headers["Profile-Title"]; len(val) > 0 {
		metadata.Title = val[0]
	}

	if val := headers["Content-Disposition"]; len(val) > 0 {
		// Try to extract filename
		if strings.Contains(val[0], "filename=") {
			parts := strings.Split(val[0], "filename=")
			if len(parts) > 1 {
				metadata.Filename = strings.Trim(parts[1], "\"")
			}
		}
	}

	return metadata
}

// SubscriptionMetadata represents metadata from subscription response
type SubscriptionMetadata struct {
	UserInfo       string              // User info (traffic, expiry, etc.)
	UpdateInterval string              // Recommended update interval
	Title          string              // Profile title
	Filename       string              // Suggested filename
	Headers        map[string][]string // All headers
}
