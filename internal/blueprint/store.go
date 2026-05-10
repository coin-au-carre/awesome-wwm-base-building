package blueprint

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadFile reads blueprint data from a JSON file.
func LoadFile(path string) ([]Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var blueprints []Blueprint
	if err := json.Unmarshal(data, &blueprints); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return blueprints, nil
}

// SaveFile writes blueprint data to a JSON file.
func SaveFile(path string, blueprints []Blueprint) error {
	data, err := json.MarshalIndent(blueprints, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling blueprints: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
