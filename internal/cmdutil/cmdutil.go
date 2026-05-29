package cmdutil

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// RequireEnv returns the value of the environment variable key,
// or logs an error and exits if it is unset.
func RequireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

// RootDir returns the repository root directory.
// It resolves to "." when run from the repo root, or ".." otherwise.
func RootDir() string {
	if _, err := os.Stat("LICENSE"); err == nil {
		return "."
	}
	return ".."
}

// LoadEnv loads the .env file from root, silently continuing if absent.
func LoadEnv(root string) {
	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}
}

// UpdateNavVersion bumps the version for the given nav key in data/nav-versions.json
// to the current UTC time. No-ops with a warning if the file cannot be read or written.
func UpdateNavVersion(root, key string) {
	path := filepath.Join(root, "data", "nav-versions.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("reading nav-versions.json", "err", err)
		return
	}
	var versions map[string]string
	if err := json.Unmarshal(raw, &versions); err != nil {
		slog.Warn("parsing nav-versions.json", "err", err)
		return
	}
	versions[key] = time.Now().UTC().Format("2006-01-02-1504")
	out, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		slog.Warn("marshaling nav-versions.json", "err", err)
		return
	}
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		slog.Warn("writing nav-versions.json", "err", err)
		return
	}
	slog.Info("nav version updated", "key", key)
}
