package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	if err := godotenv.Load(filepath.Join(cmdutil.RootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	forumID := cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	roleID := cmdutil.RequireEnv("BASE_BUILDER_ROLE_ID")

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating Discord session", "err", err)
		os.Exit(1)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	ch, err := session.Channel(forumID)
	if err != nil {
		slog.Error("fetching forum channel", "err", fmt.Errorf("assign-roles: %w", err))
		os.Exit(1)
	}
	discordGuildID := ch.GuildID

	guilds, err := guild.Load(cmdutil.RootDir())
	if err != nil {
		slog.Error("loading guilds", "err", err)
		os.Exit(1)
	}

	// Assign by posterDiscordId stored in guilds.json (no cache — always force).
	discord.AssignRoleByScore(session, discordGuildID, roleID, guilds, 0, nil, nil)

	// Also walk the forum directly to catch builders whose posterDiscordId is missing.
	if err := discord.AssignRoleToForumAuthors(session, forumID, roleID, nil, nil); err != nil {
		slog.Error("assigning role to forum authors", "err", err)
		os.Exit(1)
	}
}
