package discord

import (
	"log/slog"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const hexiPartyChannelID = "1516821438114304162"
const minereaBotID = "365594481594204161"
const geekMusicBotID = "971868710237274174"

var (
	hexiMuted   = map[string]bool{}
	hexiMutedMu sync.Mutex
)

// HandleHexiPartyMute server-mutes anyone who joins the Hexi Party voice channel,
// and unmutes them when they leave or reconnect to another channel.
func HandleHexiPartyMute(s *discordgo.Session, e *discordgo.VoiceStateUpdate) {
	if e.UserID == minereaBotID || e.UserID == geekMusicBotID {
		return
	}

	wasInChannel := e.BeforeUpdate != nil && e.BeforeUpdate.ChannelID == hexiPartyChannelID
	isInChannel := e.ChannelID == hexiPartyChannelID

	if isInChannel && !wasInChannel {
		hexiMutedMu.Lock()
		hexiMuted[e.UserID] = true
		hexiMutedMu.Unlock()
		if err := s.GuildMemberMute(e.GuildID, e.UserID, true); err != nil {
			slog.Warn("hexi party: failed to mute", "user", e.UserID, "err", err)
		} else {
			slog.Info("hexi party: muted", "user", e.UserID)
		}
		return
	}

	if wasInChannel && !isInChannel {
		if e.ChannelID != "" {
			// moved to another channel — unmute immediately
			hexiMutedMu.Lock()
			delete(hexiMuted, e.UserID)
			hexiMutedMu.Unlock()
			unmute(s, e.GuildID, e.UserID)
		}
		// full disconnect: keep in muted set, unmute on next reconnect
		return
	}

	// joined a non-Hexi channel — check if we owe them an unmute
	if !isInChannel && e.ChannelID != "" {
		hexiMutedMu.Lock()
		wasMuted := hexiMuted[e.UserID]
		if wasMuted {
			delete(hexiMuted, e.UserID)
		}
		hexiMutedMu.Unlock()
		if wasMuted {
			unmute(s, e.GuildID, e.UserID)
		}
	}
}

func unmute(s *discordgo.Session, guildID, userID string) {
	if err := s.GuildMemberMute(guildID, userID, false); err != nil {
		slog.Warn("hexi party: failed to unmute", "user", userID, "err", err)
	} else {
		slog.Info("hexi party: unmuted", "user", userID)
	}
}
