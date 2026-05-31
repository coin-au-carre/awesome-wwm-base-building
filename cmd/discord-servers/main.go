package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
)

func main() {
	cmdutil.LoadEnv(cmdutil.RootDir())
	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}

	guilds, err := s.UserGuilds(200, "", "", false)
	if err != nil {
		slog.Error("fetching guilds", "err", err)
		os.Exit(1)
	}

	fmt.Printf("Bot is in %d guild(s):\n\n", len(guilds))
	for _, g := range guilds {
		fmt.Printf("  %s\n  ID: %s\n", g.Name, g.ID)

		// Find any text channel to create an invite from.
		channels, err := s.GuildChannels(g.ID)
		if err != nil {
			fmt.Printf("  Invite: (could not fetch channels: %v)\n\n", err)
			continue
		}
		var channelID string
		for _, ch := range channels {
			if ch.Type == discordgo.ChannelTypeGuildText {
				channelID = ch.ID
				break
			}
		}
		if channelID == "" {
			fmt.Printf("  Invite: (no text channel found)\n\n")
			continue
		}

		invite, err := s.ChannelInviteCreate(channelID, discordgo.Invite{MaxAge: 300, MaxUses: 1, Unique: true})
		if err != nil {
			fmt.Printf("  Invite: (failed: %v)\n\n", err)
			continue
		}
		fmt.Printf("  Invite: https://discord.gg/%s  (expires in 5 min, 1 use)\n\n", invite.Code)
	}
}
