package discord

import (
	"fmt"
	"log/slog"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

func resolveUsername(s *discordgo.Session, userID string) string {
	if u, err := s.User(userID); err == nil {
		return u.Username
	}
	return userID
}

// AssignBaseBuilderRole grants the Base Builder role to a thread author.
// Safe to call repeatedly — Discord treats it as a no-op if already assigned.
func AssignBaseBuilderRole(s *discordgo.Session, guildID, userID, roleID string) {
	if roleID == "" || userID == "" || guildID == "" {
		return
	}
	if err := s.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
		slog.Warn("assigning base builder role failed", "user", userID, "err", err)
		return
	}
	slog.Info("base builder role assigned", "user", resolveUsername(s, userID), "id", userID)
}

// AssignRoleToVoters assigns roleID to users who voted on at least minVotes distinct guilds.
func AssignRoleToVoters(s *discordgo.Session, discordGuildID, roleID string, voterGuildCounts map[string]int, minVotes int) {
	qualified := 0
	for uid, count := range voterGuildCounts {
		if count >= minVotes {
			qualified++
		}
		_ = uid
	}
	slog.Info("assigning critic role to voters", "qualified", qualified, "min_votes", minVotes)

	assigned := 0
	for uid, count := range voterGuildCounts {
		if count < minVotes {
			continue
		}
		if err := s.GuildMemberRoleAdd(discordGuildID, uid, roleID); err != nil {
			slog.Warn("assigning critic role failed", "user", uid, "err", err)
			continue
		}
		slog.Info("critic role assigned", "user", resolveUsername(s, uid), "id", uid, "guilds_voted", count)
		assigned++
	}
	slog.Info("critic role assignment done", "assigned", assigned)
}

// AssignRoleByScore assigns roleID to any guild author whose Score >= minScore.
// skipUsers works the same as AssignRoleToForumAuthors — pass nil to assign everyone.
func AssignRoleByScore(s *discordgo.Session, discordGuildID, roleID string, guilds []guild.Guild, minScore int, skipUsers map[string]bool) {
	assigned := make(map[string]bool)
	for _, g := range guilds {
		userID := g.BuilderDiscordID
		if userID == "" || assigned[userID] || skipUsers[userID] {
			continue
		}
		if g.Score >= minScore {
			AssignBaseBuilderRole(s, discordGuildID, userID, roleID)
			assigned[userID] = true
		}
	}
}

// AssignRoleToForumAuthors fetches all threads in forumChannelID and assigns
// roleID to each thread's original poster, skipping any user ID in skipUsers.
// Pass nil skipUsers to assign everyone (e.g. with --force-role).
func AssignRoleToForumAuthors(s *discordgo.Session, forumChannelID, roleID string, skipUsers map[string]bool) error {
	if forumChannelID == "" || roleID == "" {
		return nil
	}
	ch, err := s.Channel(forumChannelID)
	if err != nil {
		return fmt.Errorf("fetching channel %s: %w", forumChannelID, err)
	}
	guildID := ch.GuildID

	threads, err := collectThreads(s, forumChannelID, guildID)
	if err != nil {
		return err
	}

	assigned := make(map[string]bool)
	for _, thread := range threads {
		msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
		if err != nil || len(msgs) == 0 {
			slog.Warn("fetching thread first message", "thread", thread.ID, "err", err)
			continue
		}
		authorID := msgs[0].Author.ID
		if assigned[authorID] || skipUsers[authorID] {
			continue
		}
		AssignBaseBuilderRole(s, guildID, authorID, roleID)
		assigned[authorID] = true
	}
	return nil
}
