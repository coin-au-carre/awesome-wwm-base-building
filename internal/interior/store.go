package interior

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadFile reads interior data from a JSON file.
func LoadFile(path string) ([]Interior, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var interiors []Interior
	if err := json.Unmarshal(data, &interiors); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return interiors, nil
}

// SaveFile writes interior data to a JSON file.
func SaveFile(path string, interiors []Interior) error {
	data, err := json.MarshalIndent(interiors, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling interiors: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
