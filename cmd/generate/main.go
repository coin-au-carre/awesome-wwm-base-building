package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"ruby/internal/generator"
	"ruby/internal/guild"
)

func main() {
	clean := flag.Bool("clean", false, "remove stale guild pages")
	root := flag.String("root", rootDir(), "root directory containing guilds.json and README.md")
	flag.Parse()

	guilds, err := guild.Load(*root)
	if err != nil {
		slog.Error("loading guilds", "err", err)
		os.Exit(1)
	}

	cfg := generator.Config{
		ReadmePath: filepath.Join(*root, "README.md"),
		GuildsDir:  filepath.Join(*root, "guilds"),
		Clean:      *clean,
	}

	if err := generator.Generate(guilds, cfg); err != nil {
		slog.Error("generation failed", "err", err)
		os.Exit(1)
	}

	slog.Info("done", "guilds", len(guilds))
}

func rootDir() string {
	if _, err := os.Stat("LICENSE"); err == nil {
		return "."
	}
	return ".."
}
