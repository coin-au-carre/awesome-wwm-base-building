package cmdutil

import (
	"log/slog"
	"os"
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
