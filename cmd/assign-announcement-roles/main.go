// One-shot: give every server member the WBM/WWM Announcements roles.
package main

import (
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
)

// ponytail: hardcoded, this is a single retroactive bulk-role run, not a recurring job
// Sequential on purpose, and one GuildMemberEdit (full roles list) per member
// instead of two GuildMemberRoleAdd calls: the role-add PUT endpoint shares a
// single tightly-throttled rate-limit bucket per guild in discordgo (see
// EndpointGuildMemberRole), so it can't be parallelized, and each add is its
// own request. Editing the full roles array in one PATCH halves the request
// count and lands in a separate, less strict bucket.
const (
	wbmAnnouncementsRoleID = "1523052174848954439"
	wwmAnnouncementsRoleID = "1523055775487099120"
)

func main() {
	cmdutil.LoadEnv(cmdutil.RootDir())

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

	total, edited := 0, 0
	for _, m := range members {
		total++
		newRoles := m.Roles
		for _, roleID := range []string{wbmAnnouncementsRoleID, wwmAnnouncementsRoleID} {
			if !hasRole(newRoles, roleID) {
				newRoles = append(newRoles, roleID)
			}
		}
		if len(newRoles) == len(m.Roles) {
			continue // already has both roles
		}
		if _, err := session.GuildMemberEdit(discordGuildID, m.User.ID, &discordgo.GuildMemberParams{Roles: &newRoles}); err != nil {
			slog.Warn("editing member roles", "user", m.User.ID, "err", err)
			continue
		}
		edited++
		if total%50 == 0 {
			slog.Info("progress", "members_seen", total, "of", len(members), "members_edited", edited)
		}
	}

	slog.Info("done", "members_seen", total, "members_edited", edited)
}

func hasRole(roles []string, roleID string) bool {
	for _, r := range roles {
		if r == roleID {
			return true
		}
	}
	return false
}
