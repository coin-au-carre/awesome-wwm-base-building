package generator

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ruby/internal/guild"
)

const (
	DiscordInvite   = "https://discord.gg/Qygt9u26Bn"
	ShowcaseChannel = "`#base-guild-showcase`"
	OwnerDiscordID  = "149790526076354561"
)

const (
	introStart           = "<!-- INTRO_START -->"
	introEnd             = "<!-- INTRO_END -->"
	tableStart           = "<!-- GENERATED_TABLE_START -->"
	tableEnd             = "<!-- GENERATED_TABLE_END -->"
	showcaseStart        = "<!-- TOP_SHOWCASE_START -->"
	showcaseEnd          = "<!-- TOP_SHOWCASE_END -->"
	lastUpdatedStart     = "<!-- LAST_UPDATED_START -->"
	lastUpdatedEnd       = "<!-- LAST_UPDATED_END -->"
	discordTemplateStart = "<!-- DISCORD_TEMPLATE_START -->"
	discordTemplateEnd   = "<!-- DISCORD_TEMPLATE_END -->"
)

type Config struct {
	ReadmePath string
	GuildsDir  string
	Clean      bool
}

func DefaultConfig() Config {
	return Config{
		ReadmePath: "README.md",
		GuildsDir:  "guilds",
	}
}

// Generate runs the full pipeline: sort → clean → guild pages → README injections.
func Generate(guilds []guild.Guild, cfg Config) error {
	sort.Slice(guilds, func(i, j int) bool {
		return guilds[i].Score > guilds[j].Score
	})

	if cfg.Clean {
		if err := cleanPages(guilds, cfg.GuildsDir); err != nil {
			return fmt.Errorf("cleaning guild pages: %w", err)
		}
	}

	if err := os.MkdirAll(cfg.GuildsDir, 0755); err != nil {
		return fmt.Errorf("creating guilds dir: %w", err)
	}

	for i := range guilds {
		if err := writePage(&guilds[i], cfg.GuildsDir); err != nil {
			return fmt.Errorf("generating page for %s: %w", guilds[i].Name, err)
		}
	}

	lastUpdated := fmt.Sprintf(
		"🔄 **Last synchronized:**  %s",
		time.Now().UTC().Format("January 2, 2006 at 15:04 UTC"),
	)

	injections := []struct{ start, end, content string }{
		{introStart, introEnd, fmt.Sprintf(
			"🏰 **%d guild bases** listed — [add yours on Discord!](%s)",
			len(guilds), DiscordInvite,
		)},
		{tableStart, tableEnd, buildTable(guilds)},
		{showcaseStart, showcaseEnd, buildTopShowcase(guilds)},
		{discordTemplateStart, discordTemplateEnd, buildGenericDiscordTemplate()},
		{lastUpdatedStart, lastUpdatedEnd, lastUpdated},
	}

	for _, inj := range injections {
		if err := injectBetweenMarkers(cfg.ReadmePath, inj.start, inj.end, inj.content); err != nil {
			return fmt.Errorf("injecting %s: %w", inj.start, err)
		}
	}

	return nil
}

func injectBetweenMarkers(file, startMk, endMk, content string) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	s := string(src)

	start := strings.Index(s, startMk)
	end := strings.Index(s, endMk)
	if start == -1 || end == -1 {
		return fmt.Errorf("markers %q / %q not found in %s", startMk, endMk, file)
	}
	if start > end {
		return fmt.Errorf("start marker appears after end marker in %s", file)
	}

	updated := s[:start] + startMk + "\n\n" + content + "\n" + s[end:]
	return os.WriteFile(file, []byte(updated), 0644)
}

func cleanPages(guilds []guild.Guild, dir string) error {
	known := make(map[string]bool, len(guilds))
	for _, g := range guilds {
		known[Slugify(g.Name)+".md"] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	var removed int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "index.md" {
			continue
		}
		if !known[e.Name()] {
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing %s: %w", path, err)
			}
			slog.Info("removed stale guild page", "path", path)
			removed++
		}
	}

	slog.Info("clean complete", "removed", removed)
	return nil
}
