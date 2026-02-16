package parser

import (
	"fmt"
	"raycoon/internal/storage/models"
)

// TUICParser implements Parser for TUIC protocol
type TUICParser struct{}

func (p *TUICParser) Protocol() string {
	return "tuic"
}

func (p *TUICParser) Parse(uri string) (*models.Config, error) {
	// TODO: Implement TUIC parser
	return nil, fmt.Errorf("TUIC parser not yet implemented")
}

func (p *TUICParser) Encode(config *models.Config) (string, error) {
	return "", fmt.Errorf("TUIC encoder not yet implemented")
}

func (p *TUICParser) Validate(config *models.Config) error {
	return fmt.Errorf("TUIC validator not yet implemented")
}
