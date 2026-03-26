package guild

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const filename = "guilds.json"

func Load(root string) ([]Guild, error) {
	data, err := os.ReadFile(filepath.Join(root, filename))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	var guilds []Guild
	if err := json.Unmarshal(data, &guilds); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filename, err)
	}
	return guilds, nil
}

func Save(root string, guilds []Guild) error {
	data, err := json.MarshalIndent(guilds, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling guilds: %w", err)
	}
	dest := filepath.Join(root, filename)
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}
