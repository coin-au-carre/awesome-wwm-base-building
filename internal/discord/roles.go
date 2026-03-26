package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// assignBaseBuilderRole grants the Base Builder role to a guild thread author.
// Safe to call repeatedly — Discord treats it as a no-op if already assigned.
func AssignBaseBuilderRole(s *discordgo.Session, guildID, userID, roleID string) {
	if roleID == "" || userID == "" || guildID == "" {
		return
	}
	if err := s.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
		slog.Warn("assigning base builder role failed", "user", userID, "err", err)
		return
	}
	slog.Info("base builder role assigned", "user", userID)
}
