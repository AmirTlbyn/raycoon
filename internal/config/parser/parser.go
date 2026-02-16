package parser

import (
	"fmt"
	"raycoon/internal/storage/models"
	"strings"
)

// Parser defines the interface for protocol parsers
type Parser interface {
	// Parse parses a URI into a Config
	Parse(uri string) (*models.Config, error)

	// Encode encodes a Config into a URI
	Encode(config *models.Config) (string, error)

	// Protocol returns the protocol name
	Protocol() string

	// Validate validates the config structure
	Validate(config *models.Config) error
}

// Registry manages protocol parsers
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry creates a new parser registry
func NewRegistry() *Registry {
	r := &Registry{
		parsers: make(map[string]Parser),
	}

	// Register built-in parsers
	r.Register(&VMessParser{})
	r.Register(&VLESSParser{})
	r.Register(&TrojanParser{})
	r.Register(&ShadowsocksParser{})
	r.Register(&Hysteria2Parser{})
	r.Register(&TUICParser{})
	r.Register(&WireGuardParser{})

	return r
}

// Register registers a new parser
func (r *Registry) Register(parser Parser) {
	r.parsers[strings.ToLower(parser.Protocol())] = parser
}

// Get retrieves a parser by protocol name
func (r *Registry) Get(protocol string) (Parser, bool) {
	parser, ok := r.parsers[strings.ToLower(protocol)]
	return parser, ok
}

// AutoDetect automatically detects protocol from URI and returns the appropriate parser
func (r *Registry) AutoDetect(uri string) (Parser, error) {
	uri = strings.TrimSpace(uri)

	// Extract protocol prefix
	idx := strings.Index(uri, "://")
	if idx == -1 {
		return nil, fmt.Errorf("invalid URI: missing protocol scheme")
	}

	protocol := strings.ToLower(uri[:idx])

	// Handle aliases
	switch protocol {
	case "ss":
		protocol = "shadowsocks"
	case "hy2":
		protocol = "hysteria2"
	}

	parser, ok := r.Get(protocol)
	if !ok {
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	return parser, nil
}

// Parse parses a URI using auto-detected protocol
func (r *Registry) Parse(uri string) (*models.Config, error) {
	parser, err := r.AutoDetect(uri)
	if err != nil {
		return nil, err
	}

	return parser.Parse(uri)
}

// ListProtocols returns a list of all supported protocols
func (r *Registry) ListProtocols() []string {
	protocols := make([]string, 0, len(r.parsers))
	for protocol := range r.parsers {
		protocols = append(protocols, protocol)
	}
	return protocols
}
