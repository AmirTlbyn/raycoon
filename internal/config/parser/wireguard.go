package parser

import (
	"fmt"
	"raycoon/internal/storage/models"
)

// WireGuardParser implements Parser for WireGuard protocol
type WireGuardParser struct{}

func (p *WireGuardParser) Protocol() string {
	return "wireguard"
}

func (p *WireGuardParser) Parse(uri string) (*models.Config, error) {
	// TODO: Implement WireGuard parser
	return nil, fmt.Errorf("WireGuard parser not yet implemented")
}

func (p *WireGuardParser) Encode(config *models.Config) (string, error) {
	return "", fmt.Errorf("WireGuard encoder not yet implemented")
}

func (p *WireGuardParser) Validate(config *models.Config) error {
	return fmt.Errorf("WireGuard validator not yet implemented")
}
