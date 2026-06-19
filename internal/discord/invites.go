package discord

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// InviteTracker caches invite use-counts so we can detect which invite a new member used.
type InviteTracker struct {
	mu      sync.Mutex
	uses    map[string]int             // code -> use count
	invites map[string]*discordgo.Invite // code -> full invite (for inviter info)
	guildID string
	logCh   string
}

func NewInviteTracker(guildID, logChannelID string) *InviteTracker {
	return &InviteTracker{
		uses:    make(map[string]int),
		invites: make(map[string]*discordgo.Invite),
		guildID: guildID,
		logCh:   logChannelID,
	}
}

func (t *InviteTracker) OnReady(s *discordgo.Session, _ *discordgo.Ready) {
	if t.guildID == "" {
		return
	}
	invites, err := s.GuildInvites(t.guildID)
	if err != nil {
		slog.Warn("invite tracker: failed to fetch invites", "err", err)
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, inv := range invites {
		t.uses[inv.Code] = inv.Uses
		t.invites[inv.Code] = inv
	}
	slog.Info("invite tracker: cached", "count", len(invites))
}

func (t *InviteTracker) OnInviteCreate(_ *discordgo.Session, e *discordgo.InviteCreate) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.uses[e.Code] = 0
	t.invites[e.Code] = &discordgo.Invite{Code: e.Code, Inviter: e.Inviter}
}

func (t *InviteTracker) OnInviteDelete(_ *discordgo.Session, e *discordgo.InviteDelete) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.uses, e.Code)
	delete(t.invites, e.Code)
}

func (t *InviteTracker) OnMemberAdd(bot *Bot) func(*discordgo.Session, *discordgo.GuildMemberAdd) {
	return func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		if t.logCh == "" {
			return
		}

		current, err := s.GuildInvites(t.guildID)
		if err != nil {
			slog.Warn("invite tracker: failed to fetch invites on join", "err", err)
			return
		}

		t.mu.Lock()

		currentCodes := make(map[string]bool, len(current))
		for _, inv := range current {
			currentCodes[inv.Code] = true
		}

		var used *discordgo.Invite

		// Find invite whose use count increased.
		for _, inv := range current {
			if prev, ok := t.uses[inv.Code]; ok && inv.Uses > prev {
				used = inv
			}
			t.uses[inv.Code] = inv.Uses
			t.invites[inv.Code] = inv
		}

		// Single-use invite disappears after use — detect by absence.
		if used == nil {
			for code, inv := range t.invites {
				if !currentCodes[code] {
					used = inv
					delete(t.uses, code)
					delete(t.invites, code)
					break
				}
			}
		}

		t.mu.Unlock()

		name := m.User.GlobalName
		if name == "" {
			name = m.User.Username
		}

		var msg string
		switch {
		case used != nil && used.Inviter != nil:
			inviter := used.Inviter.GlobalName
			if inviter == "" {
				inviter = used.Inviter.Username
			}
			msg = fmt.Sprintf("👋 **%s** (`%s`) joined via invite from **%s** (code: `%s`).", name, m.User.Username, inviter, used.Code)
		case used != nil:
			msg = fmt.Sprintf("👋 **%s** (`%s`) joined via invite code `%s`.", name, m.User.Username, used.Code)
		default:
			msg = fmt.Sprintf("👋 **%s** (`%s`) joined (invite unknown).", name, m.User.Username)
		}

		bot.Send(t.logCh, msg)

		code := ""
		if used != nil {
			code = used.Code
		}
		slog.Info("member joined", "user", m.User.Username, "display_name", name, "invite_code", code)
	}
}
