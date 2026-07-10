// One-shot: find every member holding a Homestead level role and record
// their highest level reached in data/homestead_members.json. Backfills
// data/users.json with identity info for anyone not already known.
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
	"ruby/internal/guild"
)

const homesteadChannelID = "1523299578860933220"
const defaultHomesteadMessageID = "1523309120248086589"
const homesteadAnnounceChannelID = "1521760524235309191"
const homesteadRankingsURL = "https://www.wherebuildersmeet.com/homestead/rankings?utm_source=discord&utm_medium=hall-of-fame"
const sinceLayout = "2006-01-02 15:04"

// Ordered lowest → highest so the last match wins as the member's highest level.
var homesteadLevelRoles = []struct {
	level  int
	roleID string
}{
	{5, "1523025455047770193"},
	{6, "1523026273712996513"},
	{7, "1523026359209824408"},
	{8, "1523026488981721088"},
	{9, "1523740416023990523"},
}

// Discord doesn't expose when a role was granted, so Since is first-observed:
// set the first time this sync sees a given level for a user, and kept
// across runs as long as the level hasn't changed.
type homesteadMember struct {
	Level      int    `json:"level"`
	Since      string `json:"since"`
	Username   string `json:"username"`
	GlobalName string `json:"globalName,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
}

func loadExisting(path string) map[string]homesteadMember {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]homesteadMember
	if err := json.Unmarshal(data, &m); err != nil {
		// A silent nil here makes every member look brand new next run,
		// which mass-announces "just reached" for the whole roster.
		slog.Error("parsing existing homestead_members.json, refusing to run", "path", path, "err", err)
		os.Exit(1)
	}
	return m
}

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

	outPath := filepath.Join(root, "data/homestead_members.json")
	existing := loadExisting(outPath)
	now := time.Now().UTC().Format(sinceLayout)

	result := make(map[string]homesteadMember)
	usersDirty := false
	var newAchievers []homesteadMember
	for _, m := range members {
		level := 0
		for _, lr := range homesteadLevelRoles {
			if hasRole(m.Roles, lr.roleID) {
				level = lr.level
			}
		}
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

		member := homesteadMember{
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

	data, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		slog.Error("marshalling homestead members", "err", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		slog.Error("writing homestead members", "err", err)
		os.Exit(1)
	}
	slog.Info("done", "out", outPath)

	messageID := os.Getenv("HOMESTEAD_MESSAGE_ID")
	if messageID == "" {
		messageID = defaultHomesteadMessageID
	}
	postHomesteadRanking(session, messageID, result)

	for _, m := range newAchievers {
		announceHomesteadLevelUp(session, m)
	}
}

func announceHomesteadLevelUp(s *discordgo.Session, m homesteadMember) {
	content := fmt.Sprintf("🎉 **%s** just reached **Homestead Level %d**!", displayName(m), m.Level)
	if _, err := s.ChannelMessageSend(homesteadAnnounceChannelID, content); err != nil {
		slog.Error("posting homestead level-up announcement", "user", displayName(m), "err", err)
	}
}

func displayName(m homesteadMember) string {
	if m.Nickname != "" {
		return m.Nickname
	}
	if m.GlobalName != "" {
		return m.GlobalName
	}
	return m.Username
}

func parseSince(since string) (time.Time, bool) {
	t, err := time.Parse(sinceLayout, since)
	if err != nil {
		// legacy entries recorded before "since" tracked time-of-day
		t, err = time.Parse("2006-01-02", since)
		if err != nil {
			return time.Time{}, false
		}
	}
	return t, true
}

// heldDuration renders a Discord relative timestamp (e.g. "3 hours ago")
// that the client keeps live-updating on its own, so it stays accurate even
// if the sync workflow is delayed or skipped between edits.
func heldDuration(since string) string {
	t, ok := parseSince(since)
	if !ok {
		return "?"
	}
	return fmt.Sprintf("<t:%d:R>", t.Unix())
}

var levelLabel = map[int]string{
	9: "👑  Level 9 — Master Homesteaders",
	8: "🌟  Level 8",
	7: "💠  Level 7",
	6: "🔷  Level 6",
	5: "▫️  Level 5",
}

var medals = map[int]string{0: "🥇", 1: "🥈", 2: "🥉"}

// buildHomesteadEmbed renders a leaderboard embed grouped by level (highest
// first), with medals for the top 3 overall by level then tenure. An embed
// (rather than a code block) sidesteps monospace alignment entirely — the
// font renders emoji/CJK/accented names at inconsistent widths, so a fixed
// column table never lines up cleanly across the whole roster.
func buildHomesteadEmbed(members map[string]homesteadMember) *discordgo.MessageEmbed {
	var all []homesteadMember
	for _, m := range members {
		all = append(all, m)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Level != all[j].Level {
			return all[i].Level > all[j].Level
		}
		return all[i].Since < all[j].Since
	})

	byLevel := make(map[int][]homesteadMember)
	for _, m := range all {
		byLevel[m.Level] = append(byLevel[m.Level], m)
	}

	var sb strings.Builder
	rank := 0
	for _, level := range []int{9, 8, 7, 6, 5} {
		group := byLevel[level]
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "**%s**\n", levelLabel[level])
		for _, m := range group {
			bullet := medals[rank]
			if bullet == "" {
				bullet = "🌱"
			}
			fmt.Fprintf(&sb, "%s **%s** — %s\n", bullet, displayName(m), heldDuration(m.Since))
			rank++
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "[See the full rankings ↗](%s)", homesteadRankingsURL)

	return &discordgo.MessageEmbed{
		Title:       "🏡 Homestead Hall of Fame 🏡",
		Description: strings.TrimRight(sb.String(), "\n"),
		Color:       0xF5B942,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Updated"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

func postHomesteadRanking(s *discordgo.Session, messageID string, members map[string]homesteadMember) {
	embed := buildHomesteadEmbed(members)

	if messageID == "" {
		msg, err := s.ChannelMessageSendComplex(homesteadChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			slog.Error("posting homestead ranking", "err", err)
			return
		}
		slog.Info("posted homestead ranking — add this to HOMESTEAD_MESSAGE_ID env var", "messageID", msg.ID)
		return
	}

	empty := ""
	edit := discordgo.NewMessageEdit(homesteadChannelID, messageID)
	edit.Content = &empty
	edit.Embeds = &[]*discordgo.MessageEmbed{embed}
	if _, err := s.ChannelMessageEditComplex(edit); err != nil {
		slog.Error("editing homestead ranking", "err", err)
		return
	}
	slog.Info("updated homestead ranking", "messageID", messageID)
}

func hasRole(roles []string, roleID string) bool {
	for _, r := range roles {
		if r == roleID {
			return true
		}
	}
	return false
}
