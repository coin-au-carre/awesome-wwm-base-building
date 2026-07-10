// One-shot: find every member holding a Homestead level role and record
// their highest level reached in data/homestead_members.json. Backfills
// data/users.json with identity info for anyone not already known.
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	cmdutil.LoadEnv(cmdutil.RootDir())
	root := cmdutil.RootDir()

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

	var members []*discordgo.Member
	var after string
	for {
		page, err := session.GuildMembers(discordGuildID, after, 1000)
		if err != nil {
			slog.Error("fetching guild members", "err", err)
			os.Exit(1)
		}
		if len(page) == 0 {
			break
		}
		members = append(members, page...)
		after = page[len(page)-1].User.ID
		if len(page) < 1000 {
			break
		}
	}
	slog.Info("fetched members", "count", len(members))

	users, err := guild.LoadUsers(root)
	if err != nil {
		slog.Error("loading users.json", "err", err)
		os.Exit(1)
	}

	existing, err := discord.LoadHomesteadMembers(root)
	if err != nil {
		slog.Error("loading existing homestead_members.json, refusing to run", "err", err)
		os.Exit(1)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04")

	result := make(map[string]discord.HomesteadMember)
	usersDirty := false
	var newAchievers []discord.HomesteadMember
	for _, m := range members {
		level := discord.HomesteadLevelFromRoles(m.Roles)
		if level == 0 {
			continue
		}

		info, known := users[m.User.ID]
		fresh := guild.UserInfo{
			Username:   m.User.Username,
			GlobalName: m.User.GlobalName,
			Nickname:   m.Nick,
		}
		if !known || info != fresh {
			info = fresh
			users[m.User.ID] = info
			usersDirty = true
		}

		prev, hadPrev := existing[m.User.ID]
		since := now
		if hadPrev && prev.Level == level && prev.Since != "" {
			since = prev.Since
		}

		member := discord.HomesteadMember{
			Level:      level,
			Since:      since,
			Username:   info.Username,
			GlobalName: info.GlobalName,
			Nickname:   info.Nickname,
		}
		result[m.User.ID] = member

		if level >= 7 && level > prev.Level {
			newAchievers = append(newAchievers, member)
		}
	}
	slog.Info("homestead members found", "count", len(result))

	if usersDirty {
		if err := guild.SaveUsers(root, users); err != nil {
			slog.Error("saving users.json", "err", err)
			os.Exit(1)
		}
	}

	if err := discord.SaveHomesteadMembers(root, result); err != nil {
		slog.Error("saving homestead members", "err", err)
		os.Exit(1)
	}
	slog.Info("done")

	messageID := os.Getenv("HOMESTEAD_MESSAGE_ID")
	if messageID == "" {
		messageID = discord.DefaultHomesteadMessageID
	}
	discord.PostHomesteadRanking(session, messageID, result)

	for _, m := range newAchievers {
		discord.AnnounceHomesteadLevelUp(session, m)
	}
}
