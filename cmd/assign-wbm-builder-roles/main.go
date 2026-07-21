// One-shot: grant discord.WBMBuilderRoleID to every builder in
// data/builder_identities.json who has both a discordId and a
// neteaseNumberId (i.e. a real linked NetEase account) — backfills
// anyone who registered via /wwm-uid before this role existed, or whose
// live role grant (see applyWWMUIDUpdate in internal/discord/wwm_uid.go)
// failed at the time. Safe to re-run: role_assignments.json skips
// members who already have it recorded.
package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
)

func main() {
	root := cmdutil.RootDir()
	cmdutil.LoadEnv(root)

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	forumID := cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating Discord session", "err", err)
		os.Exit(1)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	ch, err := session.Channel(forumID)
	if err != nil {
		slog.Error("fetching forum channel", "err", err)
		os.Exit(1)
	}
	discordGuildID := ch.GuildID

	identities, err := discord.LoadBuilderIdentities(root)
	if err != nil {
		slog.Error("loading builder identities", "err", err)
		os.Exit(1)
	}

	roleCachePath := filepath.Join(root, "data", "role_assignments.json")
	roleCache, err := discord.LoadRoleCache(roleCachePath)
	if err != nil {
		slog.Error("loading role cache", "err", err)
		os.Exit(1)
	}

	total := 0
	for _, entry := range identities {
		if entry.DiscordID != "" && entry.NeteaseNumberID != "" {
			total++
		}
	}
	slog.Info("wbm builder role backfill starting", "candidates", total, "total_identities", len(identities))

	assigned, alreadyHad, failed, idx := 0, 0, 0, 0
	for _, entry := range identities {
		if entry.DiscordID == "" || entry.NeteaseNumberID == "" {
			continue
		}
		idx++
		hadBefore := roleCache.Has(discord.WBMBuilderRoleID, entry.DiscordID)
		slog.Info("wbm builder role: checking", "index", idx, "total", total, "builder", entry.CanonicalAlias, "discord_id", entry.DiscordID)
		discord.AssignAwesomeBuilderRole(session, discordGuildID, entry.DiscordID, discord.WBMBuilderRoleID, "wbm-builder-backfill", roleCache)
		switch {
		case hadBefore:
			alreadyHad++
		case roleCache.Has(discord.WBMBuilderRoleID, entry.DiscordID):
			assigned++
		default:
			failed++
		}
	}
	if err := roleCache.Save(); err != nil {
		slog.Error("saving role cache", "err", err)
		os.Exit(1)
	}
	slog.Info("wbm builder role backfill done", "assigned", assigned, "already_had", alreadyHad, "failed", failed, "candidates", total)
}
