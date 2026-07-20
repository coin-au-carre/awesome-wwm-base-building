package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"ruby/internal/netease"

	"github.com/bwmarrin/discordgo"
)

// pendingWWMUID holds a resolved-but-not-yet-confirmed registration per
// Discord user, so the Confirm/"Not me" buttons don't need to round-trip
// pid/hostnum/nickname through a CustomID string. ponytail: unbounded map,
// entries are tiny and get replaced/deleted per attempt — add expiry only
// if an abandoned confirmation ever turns into a real memory concern.
var (
	pendingWWMUIDMu sync.Mutex
	pendingWWMUID   = map[string]pendingWWMUIDEntry{}
)

type pendingWWMUIDEntry struct {
	canonicalAlias string
	canonicalSlug  string
	numberID       string
	pid            string
	hostnum        int
	nickname       string
}

// handleWWMUIDCommand opens a modal pre-filled with the caller's existing
// registration (if any) — see docs/builder-identity.md's Piece 2. One
// interaction covers "show me what I have on file," "let me set or change
// it," and "let me remove it" (submit with the UID field empty).
func handleWWMUIDCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	discordID := ""
	if i.Member != nil && i.Member.User != nil {
		discordID = i.Member.User.ID
	}

	var aliasValue, uidValue string
	identities, err := LoadBuilderIdentities(root)
	if err != nil {
		slog.Warn("wwm-uid: loading builder identities", "err", err)
	} else if idx := FindBuilderIdentityByDiscordID(identities, discordID); idx >= 0 {
		aliasValue = identities[idx].CanonicalAlias
		uidValue = identities[idx].NeteaseNumberID
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: wwmUIDModalID,
			Title:    "Link Your WWM Account",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "canonical_alias",
						Label:       "Builder Name (your public URL)",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Value:       aliasValue,
						Placeholder: "e.g. Hantiya",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "uid",
						Label:       "Your In-Game UID",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Value:       uidValue,
						Placeholder: "e.g. 2039668966 — leave blank to remove",
					},
				}},
			},
		},
	})
	if err != nil {
		slog.Error("responding with wwm-uid modal", "err", err)
	}
}

func handleWWMUIDModal(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, root, devChannelID string) {
	if i.Member == nil || i.Member.User == nil {
		return
	}
	discordID := i.Member.User.ID
	fields := modalFields(i.ModalSubmitData().Components)

	alias := strings.TrimSpace(fields["canonical_alias"])
	slug := slugify(alias)
	if slug == "" {
		respondWWMUIDMessage(s, i, "Builder name can't be empty.")
		return
	}
	uid := strings.TrimSpace(fields["uid"])

	identities, err := LoadBuilderIdentities(root)
	if err != nil {
		slog.Error("wwm-uid: loading builder identities", "err", err)
		respondWWMUIDMessage(s, i, "Something went wrong loading builder data. Please try again.")
		return
	}

	selfIdx := FindBuilderIdentityByDiscordID(identities, discordID)
	if otherIdx := FindBuilderIdentityBySlug(identities, slug); otherIdx >= 0 && otherIdx != selfIdx {
		respondWWMUIDMessage(s, i, fmt.Sprintf("The builder name **%s** is already taken — please pick a different one.", alias))
		return
	}

	var existingUID, existingPID, existingNickname string
	var existingHostnum int
	if selfIdx >= 0 {
		existingUID = identities[selfIdx].NeteaseNumberID
		existingPID = identities[selfIdx].NeteasePID
		existingHostnum = identities[selfIdx].NeteaseHostnum
		existingNickname = identities[selfIdx].IngameNickname
	}

	switch {
	case uid == "":
		// Field cleared — remove NetEase fields entirely (absent, not
		// blank; see docs/builder-identity.md on why that matters).
		msg := applyWWMUIDUpdate(root, bot, devChannelID, discordID, alias, slug, "", "", 0, "")
		respondWWMUIDMessage(s, i, msg)
	case uid == existingUID && selfIdx >= 0:
		// Unchanged — only the name/slug may have moved, skip re-resolving.
		msg := applyWWMUIDUpdate(root, bot, devChannelID, discordID, alias, slug, existingUID, existingPID, existingHostnum, existingNickname)
		respondWWMUIDMessage(s, i, msg)
	default:
		// Set/changed — resolve live and confirm before saving, so a
		// typo'd UID can't silently attach a stranger's account.
		ref, err := netease.ResolveByNumberID(uid)
		if err != nil {
			slog.Warn("wwm-uid: resolving number_id", "uid", uid, "err", err)
			respondWWMUIDMessage(s, i, fmt.Sprintf("Couldn't find a WWM account with UID `%s` — double-check the number and try again.", uid))
			return
		}

		pendingWWMUIDMu.Lock()
		pendingWWMUID[discordID] = pendingWWMUIDEntry{
			canonicalAlias: alias,
			canonicalSlug:  slug,
			numberID:       uid,
			pid:            ref.PID,
			hostnum:        ref.Hostnum,
			nickname:       ref.Nickname,
		}
		pendingWWMUIDMu.Unlock()

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Found **%s** for UID `%s`. Is this you?", ref.Nickname, uid),
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.Button{Label: "Yes, that's me", Style: discordgo.SuccessButton, CustomID: wwmUIDConfirmButton},
						discordgo.Button{Label: "Not me", Style: discordgo.DangerButton, CustomID: wwmUIDNotMeButton},
					}},
				},
			},
		})
		if err != nil {
			slog.Error("responding with wwm-uid confirmation", "err", err)
		}
	}
}

func handleWWMUIDButton(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, root, devChannelID string) {
	if i.Member == nil || i.Member.User == nil {
		return
	}
	discordID := i.Member.User.ID
	customID := i.MessageComponentData().CustomID

	pendingWWMUIDMu.Lock()
	entry, ok := pendingWWMUID[discordID]
	delete(pendingWWMUID, discordID)
	pendingWWMUIDMu.Unlock()

	if !ok {
		updateWWMUIDMessage(s, i, "This confirmation expired — run `/wwm-uid` again.")
		return
	}

	if customID == wwmUIDNotMeButton {
		updateWWMUIDMessage(s, i, "Okay, not saved. Run `/wwm-uid` again with the correct UID.")
		return
	}

	msg := applyWWMUIDUpdate(root, bot, devChannelID, discordID, entry.canonicalAlias, entry.canonicalSlug, entry.numberID, entry.pid, entry.hostnum, entry.nickname)
	updateWWMUIDMessage(s, i, msg)
}

// applyWWMUIDUpdate writes discordID's record (creating it if it doesn't
// exist yet) and commits+pushes data/builder_identities.json from the live
// bot process — same pattern as data/streaming.json, see
// docs/builder-identity.md's Piece 2. Returns the message to show the user.
func applyWWMUIDUpdate(root string, bot *Bot, devChannelID, discordID, alias, slug, numberID, pid string, hostnum int, nickname string) string {
	submitMu.Lock()
	defer submitMu.Unlock()

	identities, err := LoadBuilderIdentities(root)
	if err != nil {
		slog.Error("wwm-uid: reloading builder identities before save", "err", err)
		return "Something went wrong saving your info. Please try again."
	}

	idx := FindBuilderIdentityByDiscordID(identities, discordID)
	if idx < 0 {
		identities = append(identities, BuilderIdentity{DiscordID: discordID})
		idx = len(identities) - 1
	}
	identities[idx].CanonicalAlias = alias
	identities[idx].CanonicalSlug = slug
	identities[idx].NeteaseNumberID = numberID
	identities[idx].NeteasePID = pid
	identities[idx].NeteaseHostnum = hostnum
	identities[idx].IngameNickname = nickname

	if err := SaveBuilderIdentities(root, identities); err != nil {
		slog.Error("wwm-uid: saving builder identities", "err", err)
		return "Something went wrong saving your info. Please try again."
	}

	go GitCommitAndPush(root, "data/builder_identities.json", "data: /wwm-uid "+slug, bot, devChannelID)

	if numberID == "" {
		return fmt.Sprintf("✅ Saved! Builder name: **%s**. UID removed.", alias)
	}
	return fmt.Sprintf("✅ Saved! Builder name: **%s**, UID: `%s` (**%s**).", alias, numberID, nickname)
}

func respondWWMUIDMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("responding to wwm-uid", "err", err)
	}
}

// updateWWMUIDMessage edits the confirmation message in place (removing the
// buttons) rather than posting a new message, since this is always a
// response to one of the Confirm/"Not me" buttons on that same message.
func updateWWMUIDMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{},
		},
	})
	if err != nil {
		slog.Error("updating wwm-uid confirmation message", "err", err)
	}
}
