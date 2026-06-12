package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	cmdutil.LoadEnv(cmdutil.RootDir())

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	forumID := cmdutil.RequireEnv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")
	roleID := cmdutil.RequireEnv("SOLO_BUILDER_ROLE_ID")

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating Discord session", "err", err)
		os.Exit(1)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	ch, err := session.Channel(forumID)
	if err != nil {
		slog.Error("fetching forum channel", "err", fmt.Errorf("assign-solo-roles: %w", err))
		os.Exit(1)
	}
	discordGuildID := ch.GuildID

	root := cmdutil.RootDir()
	solos, err := guild.LoadFile(filepath.Join(root, "data", "solos.json"))
	if err != nil {
		slog.Error("loading solos", "err", err)
		os.Exit(1)
	}

	// Assign by posterDiscordId stored in solos.json (no cache — always force).
	discord.AssignRoleByScore(session, discordGuildID, roleID, solos, 0, nil, nil)

	// Also walk the forum directly to catch builders whose posterDiscordId is missing.
	if err := discord.AssignRoleToForumAuthors(session, forumID, roleID, nil, nil); err != nil {
		slog.Error("assigning role to forum authors", "err", err)
		os.Exit(1)
	}
}
