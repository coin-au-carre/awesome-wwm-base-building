package discord

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
)

const (
	HomesteadChannelID         = "1523299578860933220"
	DefaultHomesteadMessageID  = "1523309120248086589"
	HomesteadAnnounceChannelID = "1521760524235309191"
	HomesteadRankingsURL       = "https://www.wherebuildersmeet.com/homestead/rankings?utm_source=discord&utm_medium=hall-of-fame"
	homesteadSinceLayout       = "2006-01-02 15:04"
)

// HomesteadLevelRoles is ordered lowest → highest so the last match wins as
// a member's highest level.
var HomesteadLevelRoles = []struct {
	Level  int
	RoleID string
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
type HomesteadMember struct {
	Level      int    `json:"level"`
	Since      string `json:"since"`
	Username   string `json:"username"`
	GlobalName string `json:"globalName,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
}

// HomesteadLevelFromRoles returns the highest homestead level implied by a
// member's role list, or 0 if they hold none of the level roles.
func HomesteadLevelFromRoles(roles []string) int {
	level := 0
	for _, lr := range HomesteadLevelRoles {
		if hasRole(roles, lr.RoleID) {
			level = lr.Level
		}
	}
	return level
}

func hasRole(roles []string, roleID string) bool {
	for _, r := range roles {
		if r == roleID {
			return true
		}
	}
	return false
}

func homesteadMembersPath(root string) string {
	return filepath.Join(root, "data/homestead_members.json")
}

// LoadHomesteadMembers reads data/homestead_members.json. A missing file
// yields an empty map; a corrupt file returns an error since silently
// treating everyone as brand new would mass-announce "just reached" for the
// whole roster.
func LoadHomesteadMembers(root string) (map[string]HomesteadMember, error) {
	data, err := os.ReadFile(homesteadMembersPath(root))
	if os.IsNotExist(err) {
		return map[string]HomesteadMember{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading homestead_members.json: %w", err)
	}
	var m map[string]HomesteadMember
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing homestead_members.json: %w", err)
	}
	return m, nil
}

func SaveHomesteadMembers(root string, members map[string]HomesteadMember) error {
	data, err := json.MarshalIndent(members, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling homestead members: %w", err)
	}
	if err := os.WriteFile(homesteadMembersPath(root), data, 0o644); err != nil {
		return fmt.Errorf("writing homestead_members.json: %w", err)
	}
	return nil
}

func HomesteadDisplayName(m HomesteadMember) string {
	if m.Nickname != "" {
		return m.Nickname
	}
	if m.GlobalName != "" {
		return m.GlobalName
	}
	return m.Username
}

func AnnounceHomesteadLevelUp(s *discordgo.Session, m HomesteadMember) {
	content := fmt.Sprintf("🎉 **%s** just reached **Homestead Level %d**!", HomesteadDisplayName(m), m.Level)
	if _, err := s.ChannelMessageSend(HomesteadAnnounceChannelID, content); err != nil {
		slog.Error("posting homestead level-up announcement", "user", HomesteadDisplayName(m), "err", err)
	}
}

func parseHomesteadSince(since string) (time.Time, bool) {
	t, err := time.Parse(homesteadSinceLayout, since)
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
	t, ok := parseHomesteadSince(since)
	if !ok {
		return "?"
	}
	return fmt.Sprintf("<t:%d:R>", t.Unix())
}

var homesteadLevelLabel = map[int]string{
	9: "👑  Level 9 — Master Homesteaders",
	8: "🌟  Level 8",
	7: "💠  Level 7",
	6: "🔷  Level 6",
	5: "▫️  Level 5",
}

var homesteadMedals = map[int]string{0: "🥇", 1: "🥈", 2: "🥉"}

// BuildHomesteadEmbed renders a leaderboard embed grouped by level (highest
// first), with medals for the top 3 overall by level then tenure. An embed
// (rather than a code block) sidesteps monospace alignment entirely — the
// font renders emoji/CJK/accented names at inconsistent widths, so a fixed
// column table never lines up cleanly across the whole roster.
func BuildHomesteadEmbed(members map[string]HomesteadMember) *discordgo.MessageEmbed {
	var all []HomesteadMember
	for _, m := range members {
		all = append(all, m)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Level != all[j].Level {
			return all[i].Level > all[j].Level
		}
		return all[i].Since < all[j].Since
	})

	byLevel := make(map[int][]HomesteadMember)
	for _, m := range all {
		byLevel[m.Level] = append(byLevel[m.Level], m)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "[See the full rankings ↗](%s)\n\n", HomesteadRankingsURL)
	rank := 0
	for _, level := range []int{9, 8, 7, 6, 5} {
		group := byLevel[level]
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "**%s**\n", homesteadLevelLabel[level])
		for _, m := range group {
			bullet := homesteadMedals[rank]
			if bullet == "" {
				bullet = "🌱"
			}
			fmt.Fprintf(&sb, "%s **%s** — %s\n", bullet, HomesteadDisplayName(m), heldDuration(m.Since))
			rank++
		}
		sb.WriteString("\n")
	}

	return &discordgo.MessageEmbed{
		Title:       "🏡 Homestead Hall of Fame 🏡",
		Description: strings.TrimRight(sb.String(), "\n"),
		Color:       0xF5B942,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Updated"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

func PostHomesteadRanking(s *discordgo.Session, messageID string, members map[string]HomesteadMember) {
	embed := BuildHomesteadEmbed(members)

	if messageID == "" {
		msg, err := s.ChannelMessageSendComplex(HomesteadChannelID, &discordgo.MessageSend{
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
	edit := discordgo.NewMessageEdit(HomesteadChannelID, messageID)
	edit.Content = &empty
	edit.Embeds = &[]*discordgo.MessageEmbed{embed}
	if _, err := s.ChannelMessageEditComplex(edit); err != nil {
		slog.Error("editing homestead ranking", "err", err)
		return
	}
	slog.Info("updated homestead ranking", "messageID", messageID)
}
