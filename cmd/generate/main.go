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
	clean := flag.Bool("clean", false, "remove stale guild/solo pages")
	root := flag.String("root", rootDir(), "root directory containing guilds.json, solos.json, and SHOWCASE.md")
	flag.Parse()

	// ── Guilds ────────────────────────────────────────────────────────────────

	guilds, err := guild.LoadFile(filepath.Join(*root, "guilds.json"))
	if err != nil {
		slog.Error("loading guilds", "err", err)
		os.Exit(1)
	}

	if err := generator.Generate(guilds, generator.Config{
		ReadmePath:  filepath.Join(*root, "SHOWCASE.md"),
		GuildsDir:   filepath.Join(*root, "guilds"),
		PagesSubdir: "guilds",
		Clean:       *clean,
	}); err != nil {
		slog.Error("guild generation failed", "err", err)
		os.Exit(1)
	}
	slog.Info("guilds done", "count", len(guilds))

	// ── Solos ─────────────────────────────────────────────────────────────────

	solosPath := filepath.Join(*root, "solos.json")
	if _, err := os.Stat(solosPath); os.IsNotExist(err) {
		slog.Info("solos.json not found, skipping solo generation")
		return
	}

	solos, err := guild.LoadFile(solosPath)
	if err != nil {
		slog.Error("loading solos", "err", err)
		os.Exit(1)
	}

	if err := generator.Generate(solos, generator.Config{
		ReadmePath:  filepath.Join(*root, "SOLO_SHOWCASE.md"),
		GuildsDir:   filepath.Join(*root, "solos"),
		PagesSubdir: "solos",
		Clean:       *clean,
	}); err != nil {
		slog.Error("solo generation failed", "err", err)
		os.Exit(1)
	}
	slog.Info("solos done", "count", len(solos))
}

func rootDir() string {
	if _, err := os.Stat("LICENSE"); err == nil {
		return "."
	}
	return ".."
}
