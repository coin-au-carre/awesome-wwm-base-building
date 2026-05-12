package cmdutil

import (
	"log/slog"
	"os"
	"path/filepath"

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
