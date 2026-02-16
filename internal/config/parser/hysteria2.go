package parser

import (
	"fmt"
	"raycoon/internal/storage/models"
)

// Hysteria2Parser implements Parser for Hysteria2 protocol
type Hysteria2Parser struct{}

func (p *Hysteria2Parser) Protocol() string {
	return "hysteria2"
}

func (p *Hysteria2Parser) Parse(uri string) (*models.Config, error) {
	// TODO: Implement Hysteria2 parser
	return nil, fmt.Errorf("Hysteria2 parser not yet implemented")
}

func (p *Hysteria2Parser) Encode(config *models.Config) (string, error) {
	return "", fmt.Errorf("Hysteria2 encoder not yet implemented")
}

func (p *Hysteria2Parser) Validate(config *models.Config) error {
	return fmt.Errorf("Hysteria2 validator not yet implemented")
}
