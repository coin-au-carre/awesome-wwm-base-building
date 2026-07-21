package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"ruby/internal/guild"
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
	aliases        []string
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

	// Default to their Discord nickname for first-time registration — just
	// a starting suggestion, still fully editable, not an identity rule
	// (see docs/builder-identity.md: canonicalSlug intentionally stays
	// independent of Discord nickname, which can differ from how a
	// builder's name is actually credited in guild/solo/blueprint data).
	aliasValue := memberDisplayName(i)
	var uidValue, aliasesValue string
	uidLabel := "Your In-Game UID"
	identities, err := LoadBuilderIdentities(root)
	if err != nil {
		slog.Warn("wwm-uid: loading builder identities", "err", err)
	} else if idx := FindBuilderIdentityByDiscordID(identities, discordID); idx >= 0 {
		aliasValue = identities[idx].CanonicalAlias
		uidValue = identities[idx].NeteaseNumberID
		aliasesValue = strings.Join(identities[idx].Aliases, ", ")
		// Discord modal labels cap at 45 chars — this is display-only info,
		// not an input, so truncate rather than error if the nickname is long.
		if nick := identities[idx].IngameNickname; nick != "" {
			label := fmt.Sprintf("Your In-Game UID (currently: %s)", nick)
			if len(label) > 45 {
				label = label[:44] + "…"
			}
			uidLabel = label
		}
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
						Label:       uidLabel,
						Style:       discordgo.TextInputShort,
						Required:    false,
						Value:       uidValue,
						Placeholder: "e.g. 2039668966 — leave blank to remove",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "aliases",
						Label:       "Other Names/Aliases (optional)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Value:       aliasesValue,
						Placeholder: "comma-separated, e.g. OldName, Nickname2",
					},
				}},
			},
		},
	})
	if err != nil {
		slog.Error("responding with wwm-uid modal", "err", err)
	}
}

// parseAliases splits a comma-separated field into trimmed, non-empty names.
func parseAliases(raw string) []string {
	var out []string
	for _, a := range strings.Split(raw, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

// conflictingAlias returns the first alias in aliases whose slug collides
// with another identity's canonicalSlug or alias (not selfIdx's own), or ""
// if none conflict.
func conflictingAlias(identities []BuilderIdentity, selfIdx int, aliases []string) string {
	for _, a := range aliases {
		aSlug := slugify(a)
		for idx, entry := range identities {
			if idx == selfIdx {
				continue
			}
			if entry.CanonicalSlug == aSlug {
				return a
			}
			for _, existingAlias := range entry.Aliases {
				if slugify(existingAlias) == aSlug {
					return a
				}
			}
		}
	}
	return ""
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
		respondWWMUIDMessage(s, i, bot, devChannelID, discordID, "❌ Builder name can't be empty.")
		return
	}
	uid := strings.TrimSpace(fields["uid"])
	aliases := parseAliases(fields["aliases"])

	identities, err := LoadBuilderIdentities(root)
	if err != nil {
		slog.Error("wwm-uid: loading builder identities", "err", err)
		respondWWMUIDMessage(s, i, bot, devChannelID, discordID, "❌ Something went wrong loading builder data. Please try again.")
		return
	}

	selfIdx := FindBuilderIdentityByDiscordID(identities, discordID)
	otherIdx := FindBuilderIdentityBySlug(identities, slug)
	// A slug can belong to an unclaimed stub entry — one auto-created from
	// a guild/solo/blueprint credit match (see the getStaticPaths loop in
	// builders/[slug].astro) that has no DiscordID attached yet. That's not
	// "taken" by anyone, it's just this builder's name waiting for them to
	// actually run /wwm-uid — let them claim it instead of rejecting.
	claiming := otherIdx >= 0 && otherIdx != selfIdx && identities[otherIdx].DiscordID == ""
	if otherIdx >= 0 && otherIdx != selfIdx && !claiming {
		respondWWMUIDMessage(s, i, bot, devChannelID, discordID, fmt.Sprintf("❌ The builder name **%s** is already taken — please pick a different one.", alias))
		return
	}
	if bad := conflictingAlias(identities, selfIdx, aliases); bad != "" {
		respondWWMUIDMessage(s, i, bot, devChannelID, discordID, fmt.Sprintf("❌ The alias **%s** is already used by another builder — please remove it.", bad))
		return
	}

	sourceIdx := selfIdx
	if sourceIdx < 0 && claiming {
		sourceIdx = otherIdx
	}
	var existingUID, existingPID, existingNickname string
	var existingHostnum int
	if sourceIdx >= 0 {
		existingUID = identities[sourceIdx].NeteaseNumberID
		existingPID = identities[sourceIdx].NeteasePID
		existingHostnum = identities[sourceIdx].NeteaseHostnum
		existingNickname = identities[sourceIdx].IngameNickname
	}

	switch {
	case uid == "":
		// Field cleared — remove NetEase fields entirely (absent, not
		// blank; see docs/builder-identity.md on why that matters).
		msg := applyWWMUIDUpdate(s, root, bot, devChannelID, i, alias, slug, aliases, "", "", 0, "")
		respondWWMUIDMessage(s, i, bot, devChannelID, discordID, msg)
	case uid == existingUID && sourceIdx >= 0:
		// Unchanged — only the name/slug/aliases may have moved, skip re-resolving.
		msg := applyWWMUIDUpdate(s, root, bot, devChannelID, i, alias, slug, aliases, existingUID, existingPID, existingHostnum, existingNickname)
		respondWWMUIDMessage(s, i, bot, devChannelID, discordID, msg)
	default:
		// Set/changed — resolve live and confirm before saving, so a
		// typo'd UID can't silently attach a stranger's account.
		ref, err := netease.ResolveByNumberID(uid)
		if err != nil {
			slog.Warn("wwm-uid: resolving number_id", "uid", uid, "err", err)
			respondWWMUIDMessage(s, i, bot, devChannelID, discordID, fmt.Sprintf("❌ Couldn't find a WWM account with UID `%s` — double-check the number and try again.", uid))
			return
		}

		pendingWWMUIDMu.Lock()
		pendingWWMUID[discordID] = pendingWWMUIDEntry{
			canonicalAlias: alias,
			canonicalSlug:  slug,
			aliases:        aliases,
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
		updateWWMUIDMessage(s, i, bot, devChannelID, discordID, "This confirmation expired — run `/wwm-uid` again.")
		return
	}

	if customID == wwmUIDNotMeButton {
		updateWWMUIDMessage(s, i, bot, devChannelID, discordID, "Okay, not saved. Run `/wwm-uid` again with the correct UID.")
		return
	}

	msg := applyWWMUIDUpdate(s, root, bot, devChannelID, i, entry.canonicalAlias, entry.canonicalSlug, entry.aliases, entry.numberID, entry.pid, entry.hostnum, entry.nickname)
	updateWWMUIDMessage(s, i, bot, devChannelID, discordID, msg)
}

// applyWWMUIDUpdate writes discordID's record (creating it if it doesn't
// exist yet) and commits+pushes data/builder_identities.json from the live
// bot process — same pattern as data/streaming.json, see
// docs/builder-identity.md's Piece 2. Also upserts data/discord_users.json
// with the member's current username/globalName/nickname — normally only
// backfilled by cmd/homestead-sync, but the member info is already in hand
// here for free, so a builder registering via /wwm-uid before ever earning
// a Homestead role doesn't have to wait for the next sync to show up in
// name-based lookups (e.g. the "Scouted" section on /builders/<slug>).
// Also grants/revokes WBMBuilderRoleID depending on whether a UID is set —
// mirrors cmd/assign-wbm-builder-roles' backfill logic for existing
// entries. Returns the message to show the user.
func applyWWMUIDUpdate(s *discordgo.Session, root string, bot *Bot, devChannelID string, i *discordgo.InteractionCreate, alias, slug string, aliases []string, numberID, pid string, hostnum int, nickname string) string {
	submitMu.Lock()
	defer submitMu.Unlock()

	discordID := i.Member.User.ID

	identities, err := LoadBuilderIdentities(root)
	if err != nil {
		slog.Error("wwm-uid: reloading builder identities before save", "err", err)
		return "❌ Something went wrong saving your info. Please try again."
	}

	idx := FindBuilderIdentityByDiscordID(identities, discordID)
	if idx < 0 {
		// No entry for this Discord ID yet — if the slug belongs to an
		// unclaimed stub (no DiscordID, just an auto-created guild/solo
		// credit match), claim it instead of creating a duplicate
		// canonicalSlug entry (see the "claiming" check in
		// handleWWMUIDModal, which already allowed this case through).
		if claimIdx := FindBuilderIdentityBySlug(identities, slug); claimIdx >= 0 && identities[claimIdx].DiscordID == "" {
			idx = claimIdx
			identities[idx].DiscordID = discordID
		} else {
			identities = append(identities, BuilderIdentity{DiscordID: discordID})
			idx = len(identities) - 1
		}
	}
	identities[idx].CanonicalAlias = alias
	identities[idx].CanonicalSlug = slug
	identities[idx].Aliases = aliases
	identities[idx].NeteaseNumberID = numberID
	identities[idx].NeteasePID = pid
	identities[idx].NeteaseHostnum = hostnum
	identities[idx].IngameNickname = nickname

	if err := SaveBuilderIdentities(root, identities); err != nil {
		slog.Error("wwm-uid: saving builder identities", "err", err)
		return "❌ Something went wrong saving your info. Please try again."
	}

	files := []string{"data/builder_identities.json"}
	if usersDirty := upsertDiscordUser(root, i); usersDirty {
		files = append(files, "data/discord_users.json")
	}
	go GitCommitAndPush(root, "data: /wwm-uid "+slug, bot, devChannelID, files...)

	// Best-effort — a role sync failure shouldn't block the save itself,
	// cmd/assign-wbm-builder-roles can always backfill it later.
	if numberID != "" {
		if err := s.GuildMemberRoleAdd(i.GuildID, discordID, WBMBuilderRoleID); err != nil {
			slog.Warn("wwm-uid: granting WBM Builder role", "user", discordID, "err", err)
		}
	} else {
		if err := s.GuildMemberRoleRemove(i.GuildID, discordID, WBMBuilderRoleID); err != nil {
			slog.Warn("wwm-uid: revoking WBM Builder role", "user", discordID, "err", err)
		}
	}

	if numberID == "" {
		return fmt.Sprintf("✅ Saved! Builder name: **%s**. UID removed.", alias)
	}
	return fmt.Sprintf("✅ Saved! Builder name: **%s**, UID: `%s` (**%s**).", alias, numberID, nickname)
}

// upsertDiscordUser records/refreshes discordID's username/globalName/
// nickname in data/discord_users.json, same fresh-vs-known diff pattern as
// cmd/homestead-sync. Returns whether the file changed (and was saved).
func upsertDiscordUser(root string, i *discordgo.InteractionCreate) bool {
	discordID := i.Member.User.ID
	users, err := guild.LoadUsers(root)
	if err != nil {
		slog.Warn("wwm-uid: loading discord_users.json", "err", err)
		return false
	}

	fresh := guild.UserInfo{
		Username:   i.Member.User.Username,
		GlobalName: i.Member.User.GlobalName,
		Nickname:   i.Member.Nick,
	}
	if existing, known := users[discordID]; known && existing == fresh {
		return false
	}
	users[discordID] = fresh

	if err := guild.SaveUsers(root, users); err != nil {
		slog.Warn("wwm-uid: saving discord_users.json", "err", err)
		return false
	}
	return true
}

// respondWWMUIDMessage replies to the modal submission and mirrors the
// outcome to devChannelID, so mods can see /wwm-uid successes/failures
// without needing to be the user hitting them.
func respondWWMUIDMessage(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, devChannelID, discordID, content string) {
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
	logWWMUIDOutcome(bot, devChannelID, discordID, content)
}

// updateWWMUIDMessage edits the confirmation message in place (removing the
// buttons) rather than posting a new message, since this is always a
// response to one of the Confirm/"Not me" buttons on that same message. Also
// mirrors the outcome to devChannelID, same as respondWWMUIDMessage.
func updateWWMUIDMessage(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, devChannelID, discordID, content string) {
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
	logWWMUIDOutcome(bot, devChannelID, discordID, content)
}

func logWWMUIDOutcome(bot *Bot, devChannelID, discordID, content string) {
	if devChannelID == "" {
		return
	}
	bot.Send(devChannelID, fmt.Sprintf("🪪 `/wwm-uid` <@%s>: %s", discordID, content))
}
