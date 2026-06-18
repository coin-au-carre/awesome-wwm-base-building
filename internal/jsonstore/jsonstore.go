package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
)

func Load[T any](path string) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return items, nil
}

func Save[T any](path string, items []T) error {
	data, err := json.MarshalIndent(items, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
