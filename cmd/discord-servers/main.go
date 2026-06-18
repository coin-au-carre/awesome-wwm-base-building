package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
)

const protectedGuildID = "1483447710617960508" // Where Builders Meet

func main() {
	leaveID := flag.String("leave", "", "leave the Discord server with this ID")
	flag.Parse()

	cmdutil.LoadEnv(cmdutil.RootDir())
	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}

	if *leaveID != "" {
		if *leaveID == protectedGuildID {
			slog.Error("refusing to leave protected server", "id", *leaveID)
			os.Exit(1)
		}
		if err := s.GuildLeave(*leaveID); err != nil {
			slog.Error("leaving server", "id", *leaveID, "err", err)
			os.Exit(1)
		}
		slog.Info("left server", "id", *leaveID)
		return
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
