package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

const hexiPartyChannelID = "1516821438114304162"

// HandleHexiPartyMute server-mutes anyone who joins the Hexi Party voice channel,
// and unmutes them when they leave.
func HandleHexiPartyMute(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	wasInChannel := e.BeforeUpdate != nil && e.BeforeUpdate.ChannelID == hexiPartyChannelID
	isInChannel := e.ChannelID == hexiPartyChannelID

	if isInChannel == wasInChannel {
		return
	}

	if isInChannel {
		if err := s.GuildMemberMute(e.GuildID, e.UserID, true); err != nil {
			slog.Warn("hexi party: failed to mute", "user", e.UserID, "err", err)
		} else {
			slog.Info("hexi party: muted", "user", e.UserID)
		}
		return
	}

	// Only unmute when moving to another channel; Discord clears the mute automatically on full disconnect.
	if wasInChannel && e.ChannelID != "" {
		if err := s.GuildMemberMute(e.GuildID, e.UserID, false); err != nil {
			slog.Warn("hexi party: failed to unmute", "user", e.UserID, "err", err)
		} else {
			slog.Info("hexi party: unmuted", "user", e.UserID)
		}
	}
}
