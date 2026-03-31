package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

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
	username := userID
	if u, err := s.User(userID); err == nil {
		username = u.Username
	}
	slog.Info("base builder role assigned", "user", username)
}

// AssignRoleToForumAuthors fetches all threads in forumChannelID and assigns
// roleID to each thread's original poster. Safe to call repeatedly.
func AssignRoleToForumAuthors(s *discordgo.Session, forumChannelID, roleID string) error {
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
		if assigned[authorID] {
			continue
		}
		AssignBaseBuilderRole(s, guildID, authorID, roleID)
		assigned[authorID] = true
	}
	return nil
}
