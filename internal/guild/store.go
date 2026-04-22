package guild

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const ModifiedLayout = "January 2, 2006 at 03:04 PM UTC"

func ModifiedNow() string {
	return time.Now().UTC().Format(ModifiedLayout)
}

const filename = "data/guilds.json"

func Load(root string) ([]Guild, error) {
	return LoadFile(filepath.Join(root, filename))
}

func Save(root string, guilds []Guild) error {
	return SaveFile(filepath.Join(root, filename), guilds)
}

// LoadFile reads guild data from an arbitrary JSON file path.
func LoadFile(path string) ([]Guild, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var guilds []Guild
	if err := json.Unmarshal(data, &guilds); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return guilds, nil
}

// LoadVoterBlacklist reads a JSON array of user IDs to exclude from scoring.
// Returns an empty set (not an error) if the file doesn't exist.
func LoadVoterBlacklist(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// SaveFile writes guild data to an arbitrary JSON file path.
func SaveFile(path string, guilds []Guild) error {
	data, err := json.MarshalIndent(guilds, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling guilds: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
