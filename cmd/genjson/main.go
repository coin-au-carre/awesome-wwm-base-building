package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"ruby/internal/cmdutil"
	"ruby/internal/guild"
)

// publicGuild mirrors Guild but omits the Note field.
type publicGuild struct {
	ID                 string                    `json:"id,omitempty"`
	Name               string                    `json:"name"`
	GuildName          string                    `json:"guildName,omitempty"`
	Builders           []string                  `json:"builders"`
	Tags               []string                  `json:"tags,omitempty"`
	DiscordThread      string                    `json:"discordThread"`
	BuilderDiscordID   string                    `json:"builderDiscordId,omitempty"`
	Lore               string                    `json:"lore,omitempty"`
	WhatToVisit        string                    `json:"whatToVisit,omitempty"`
	Score              int                       `json:"score"`
	CoverImage         string                    `json:"coverImage,omitempty"`
	Screenshots        []string                  `json:"screenshots,omitempty"`
	ScreenshotSections []guild.ScreenshotSection `json:"screenshotSections,omitempty"`
	Videos             []string                  `json:"videos,omitempty"`
	LastModified       string                    `json:"lastModified,omitempty"`
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	out := flag.String("out", "", "output path (default: <root>/web/public/guilds.json)")
	flag.Parse()

	if *out == "" {
		*out = filepath.Join(*root, "web", "public", "guilds.json")
	}

	guilds, err := guild.Load(*root)
	if err != nil {
		slog.Error("loading guilds", "err", err)
		os.Exit(1)
	}

	public := make([]publicGuild, len(guilds))
	for i, g := range guilds {
		public[i] = publicGuild{
			ID:                 g.ID,
			Name:               g.Name,
			GuildName:          g.GuildName,
			Builders:           g.Builders,
			Tags:               g.Tags,
			DiscordThread:      g.DiscordThread,
			BuilderDiscordID:   g.BuilderDiscordID,
			Lore:               g.Lore,
			WhatToVisit:        g.WhatToVisit,
			Score:              g.Score,
			CoverImage:         g.CoverImage,
			Screenshots:        g.Screenshots,
			ScreenshotSections: g.ScreenshotSections,
			Videos:             g.Videos,
			LastModified:       g.LastModified,
		}
	}

	data, err := json.MarshalIndent(public, "", "\t")
	if err != nil {
		slog.Error("marshalling", "err", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, data, 0644); err != nil {
		slog.Error("writing file", "path", *out, "err", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d guilds to %s\n", len(public), *out)
}
