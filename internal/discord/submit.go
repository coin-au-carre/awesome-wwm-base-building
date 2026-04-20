package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const (
	submitCommandName = "scout-guild"
	submitModalID     = "scout_guild_modal"

	postCommandName = "submit-guild"
	postModalID     = "submit_guild_modal"

	soloCommandName = "submit-solo"
	soloModalID     = "submit_solo_modal"

	welcomeTestCommandName = "welcome-test"
)

var submitMu sync.Mutex

var knownTags = map[string]bool{
	"Arena": true, "Cave": true, "City": true, "Creative": true,
	"Cute": true, "Dance Floor": true, "Desert": true, "Floating island": true,
	"Fun": true, "Maze": true, "Military": true, "Mountain": true,
	"Nature": true, "River": true, "Snow": true, "Zen": true,
}

var appreciationScore = map[string]int{
	"s": 2, "S": 2,
	"a": 1, "A": 1,
	"b": 0, "B": 0,
}

func RegisterSubmitCommand(s *discordgo.Session, discordGuildID string) {
	_, err := s.ApplicationCommandCreate(s.State.User.ID, discordGuildID, &discordgo.ApplicationCommand{
		Name:        submitCommandName,
		Description: "Scout a guild base and add it to the directory",
	})
	if err != nil {
		slog.Error("registering scout-guild command", "err", err)
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, discordGuildID, &discordgo.ApplicationCommand{
		Name:        postCommandName,
		Description: "Submit your guild base to the showcase",
	})
	if err != nil {
		slog.Error("registering submit-guild command", "err", err)
	}

	_, err = s.ApplicationCommandCreate(s.State.User.ID, discordGuildID, &discordgo.ApplicationCommand{
		Name:        soloCommandName,
		Description: "Submit your solo construction to the showcase",
	})
	if err != nil {
		slog.Error("registering submit-solo command", "err", err)
	}

	adminPerm := int64(discordgo.PermissionAdministrator)
	_, err = s.ApplicationCommandCreate(s.State.User.ID, discordGuildID, &discordgo.ApplicationCommand{
		Name:                     welcomeTestCommandName,
		Description:              "Preview the welcome message (admins only)",
		DefaultMemberPermissions: &adminPerm,
	})
	if err != nil {
		slog.Error("registering welcome-test command", "err", err)
	}
}

func OnInteractionCreate(bot *Bot, root, submissionChannelID, discoveriesChannelID, guildForumChannelID, soloForumChannelID string) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case submitCommandName:
				handleSubmitCommand(s, i)
			case postCommandName:
				handlePostCommand(s, i)
			case soloCommandName:
				handleSoloCommand(s, i)
			case welcomeTestCommandName:
				handleWelcomeTestCommand(s, i)
			}
		case discordgo.InteractionModalSubmit:
			switch i.ModalSubmitData().CustomID {
			case submitModalID:
				handleSubmitModal(s, i, bot, root, submissionChannelID, discoveriesChannelID)
			case postModalID:
				handlePostModal(s, i, bot, submissionChannelID, guildForumChannelID)
			case soloModalID:
				handleSoloModal(s, i, bot, submissionChannelID, soloForumChannelID)
			}
		}
	}
}

func handleSubmitCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	slog.Info("scout-guild command received", "user", i.Member.User.Username)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: submitModalID,
			Title:    "Scout Guild Base",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "name",
						Label:       "Guild Name [GuildID(optional)]",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "Iron Vanguard [12345678]",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "what_to_visit",
						Label:       "What to Visit (points of interest)",
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Placeholder: "Key spots, landmarks, must-see areas...",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "appreciation",
						Label:       "Appreciation (default: B)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "S must-see, A Great, B Nice",
						MaxLength:   1,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "tags",
						Label:       "Tags — optional, comma-separated",
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: "Arena, Cave, City, Creative, Cute, Desert, Fun, Maze, Military, Mountain, Nature, River, Snow, Zen",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "builders_proposed",
						Label:       "Have you proposed builders to join WBM?",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "yes / no (default: no)",
						MaxLength:   3,
					},
				}},
			},
		},
	})
	if err != nil {
		slog.Error("responding with modal", "err", err)
	}
}

func handleSubmitModal(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, root, submissionChannelID, discoveriesChannelID string) {
	fields := modalFields(i.ModalSubmitData().Components)

	name, guildID := parseLocation(fields["name"])
	whatToVisit := fields["what_to_visit"]
	tags := filterTags(splitCSV(fields["tags"]))
	buildersProposed := strings.ToLower(strings.TrimSpace(fields["builders_proposed"])) == "yes"

	appreciation := strings.ToUpper(strings.TrimSpace(fields["appreciation"]))
	if appreciation == "" {
		appreciation = "B"
	}
	score, valid := appreciationScore[appreciation]
	if !valid {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Appreciation must be S, A, or B.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	g := guild.Guild{
		ID:           guildID,
		Name:         name,
		WhatToVisit:  whatToVisit,
		Builders:     []string{},
		Tags:         tags,
		Score:        score,
		LastModified: guild.ModifiedNow(),
	}
	if buildersProposed {
		n := guild.Note{Text: "Builders proposed to WBM."}
		g.Note = &n
	}

	submitMu.Lock()
	guilds, err := guild.Load(root)
	if err == nil {
		guilds = append([]guild.Guild{g}, guilds...)
		err = guild.Save(root, guilds)
	}
	submitMu.Unlock()

	if err != nil {
		slog.Error("saving guild proposal", "name", name, "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Something went wrong saving the guild. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	slog.Info("guild proposed", "user", i.Member.User.Username, "name", name, "appreciation", appreciation, "score", score)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Guild **%s** scouted successfully! (appreciation: %s)", name, appreciation),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	submitter := "unknown"
	if i.Member != nil {
		submitter = i.Member.DisplayName()
	}

	jsonBytes, _ := json.MarshalIndent(g, "", "\t")
	notice := fmt.Sprintf("**New guild scouted** by %s\n```json\n%s\n```", submitter, string(jsonBytes))
	if submissionChannelID != "" {
		bot.Send(submissionChannelID, notice)
	} else {
		bot.Notify(notice)
	}

	if discoveriesChannelID != "" {
		discovery := buildDiscoveryMessage(submitter, g)
		bot.Send(discoveriesChannelID, discovery)
	}
}

const maxWhatToVisit = 80

func buildDiscoveryMessage(explorer string, g guild.Guild) string {
	title := g.Name
	if g.ID != "" {
		title = fmt.Sprintf("%s [%s]", g.Name, g.ID)
	}

	wtv := g.WhatToVisit
	if len(wtv) > maxWhatToVisit {
		wtv = wtv[:maxWhatToVisit] + " [...]"
	}

	line2 := "📍 " + wtv
	if len(g.Tags) > 0 {
		line2 += "  ·  " + strings.Join(g.Tags, " · ")
	}

	return fmt.Sprintf("🧭 **%s** — *scouted by %s*\n%s", title, explorer, line2)
}

func handlePostCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	slog.Info("submit-guild command received", "user", i.Member.User.Username)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: postModalID,
			Title:    "Submit Your Guild Base",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "name",
						Label:       "Guild Name [GuildID(optional)]",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "Iron Vanguard [12345678]",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "builders",
						Label:       "Builders — in-game names, comma-separated",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "BuilderOne, BuilderTwo",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "lore",
						Label:       "Lore — optional",
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: "The story or theme behind your base...",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "what_to_visit",
						Label:       "What to Visit — optional",
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: "- Point of interest 1\n- Point of interest 2",
					},
				}},
			},
		},
	})
	if err != nil {
		slog.Error("responding with submit-base modal", "err", err)
	}
}

func handlePostModal(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, submissionChannelID, guildForumChannelID string) {
	fields := modalFields(i.ModalSubmitData().Components)
	name, guildID := parseLocation(fields["name"])
	builders := fields["builders"]
	lore := fields["lore"]
	whatToVisit := fields["what_to_visit"]

	// Build thread title: "GuildName [ID]" or just "GuildName"
	threadTitle := name
	if guildID != "" {
		threadTitle = fmt.Sprintf("%s [%s]", name, guildID)
	}

	// Build the content template for the user to copy-paste as their own post
	var content strings.Builder
	content.WriteString(fmt.Sprintf("## 🏯 %s\n\n", threadTitle))
	if builders != "" {
		content.WriteString(fmt.Sprintf("👷 Builders: %s\n\n", builders))
	}
	if lore != "" {
		content.WriteString(fmt.Sprintf("### 📝 Lore\n%s\n\n", lore))
	}
	if whatToVisit != "" {
		content.WriteString(fmt.Sprintf("### 🧙 What to visit\n%s", whatToVisit))
	}

	slog.Info("submit-guild form received", "user", i.Member.User.Username, "name", name)

	guildURL := websiteBase + "/guilds/" + slugify(name)
	channelMention := "<#" + guildForumChannelID + ">"
	if guildForumChannelID == "" {
		channelMention = "**#guild-base-showcase**"
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Check your DMs — I sent you your formatted post to copy into %s! 📬", channelMention),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	if submissionChannelID != "" {
		bot.Send(submissionChannelID, fmt.Sprintf("**/submit-guild filled ** by %s: **%s**", i.Member.DisplayName(), threadTitle))
	}

	if ch, err := s.UserChannelCreate(i.Member.User.ID); err == nil {
		dm := fmt.Sprintf(
			"## 🏯 %s\n\n"+
				"Here's your formatted post, ready to copy!\n\n"+
				"**1.** Go to %s\n"+
				"**2.** Create a new post titled: `%s`\n"+
				"**3.** Paste the text below as your message\n"+
				"**4.** Add your screenshots 📸\n\n"+
				"```\n%s\n```\n\n"+
				"🌐 **Future page on the website:** <%s>\n"+
				"*(may take a little while to appear once everything syncs)*",
			name, channelMention, threadTitle, strings.TrimSpace(content.String()), guildURL,
		)
		_, _ = s.ChannelMessageSend(ch.ID, dm)
	}
}

func handleSoloCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	slog.Info("submit-solo command received", "user", i.Member.User.Username)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: soloModalID,
			Title:    "Submit Your Solo Construction",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "name",
						Label:       "Work label  —  Builder ID optional",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "My Build Name [12345678]",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "builders",
						Label:       "Builders — in-game names, comma-separated",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "BuilderOne, BuilderTwo",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "lore",
						Label:       "Lore — optional",
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: "The story or theme behind your build...",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "what_to_visit",
						Label:       "What to Visit — optional",
						Style:       discordgo.TextInputParagraph,
						Required:    false,
						Placeholder: "- Point of interest 1\n- Point of interest 2",
					},
				}},
			},
		},
	})
	if err != nil {
		slog.Error("responding with submit-solo modal", "err", err)
	}
}

func handleSoloModal(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, submissionChannelID, soloForumChannelID string) {
	fields := modalFields(i.ModalSubmitData().Components)
	name, buildID := parseLocation(fields["name"])
	builders := fields["builders"]
	lore := fields["lore"]
	whatToVisit := fields["what_to_visit"]

	threadTitle := name
	if buildID != "" {
		threadTitle = fmt.Sprintf("%s [%s]", name, buildID)
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("## 🏠 %s\n\n", threadTitle))
	if builders != "" {
		content.WriteString(fmt.Sprintf("👷 Builders: %s\n\n", builders))
	}
	if lore != "" {
		content.WriteString(fmt.Sprintf("### 📝 Lore\n%s\n\n", lore))
	}
	if whatToVisit != "" {
		content.WriteString(fmt.Sprintf("### 🧙 What to visit\n%s", whatToVisit))
	}

	slog.Info("submit-solo form received", "user", i.Member.User.Username, "name", name)

	channelMention := "<#" + soloForumChannelID + ">"
	if soloForumChannelID == "" {
		channelMention = "**#solo-building-showcase**"
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Check your DMs — I sent you your formatted post to copy into %s! 📬", channelMention),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	if submissionChannelID != "" {
		bot.Send(submissionChannelID, fmt.Sprintf("**New solo submission** by %s: **%s**", i.Member.DisplayName(), threadTitle))
	}

	soloURL := websiteBase + "/solos/" + slugify(name)
	if ch, err := s.UserChannelCreate(i.Member.User.ID); err == nil {
		dm := fmt.Sprintf(
			"## 🏠 %s\n\n"+
				"Here's your formatted post, ready to copy!\n\n"+
				"**1.** Go to %s\n"+
				"**2.** Create a new post titled: `%s`\n"+
				"**3.** Paste the text below as your message\n"+
				"**4.** Add your screenshots 📸\n\n"+
				"```\n%s\n```\n\n"+
				"🌐 **Future page on the website:** <%s>\n"+
				"*(may take a little while to appear once everything syncs)*",
			name, channelMention, threadTitle, strings.TrimSpace(content.String()), soloURL,
		)
		_, _ = s.ChannelMessageSend(ch.ID, dm)
	}
}

func filterTags(tags []string) []string {
	var out []string
	for _, t := range tags {
		t = titleCase(t)
		if knownTags[t] {
			out = append(out, t)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func modalFields(components []discordgo.MessageComponent) map[string]string {
	out := make(map[string]string)
	for _, row := range components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, c := range ar.Components {
			ti, ok := c.(*discordgo.TextInput)
			if !ok {
				continue
			}
			out[ti.CustomID] = strings.TrimSpace(ti.Value)
		}
	}
	return out
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

// BuildWelcomeMessage returns the welcome message for a new member.
func BuildWelcomeMessage(name string) string {
	return "**Welcome to [Where Builders Meet](<https://www.wherebuildersmeet.com>), " + name + "!** :wave:\n\n" +
		"**[Builder?](<https://www.wherebuildersmeet.com/contribute/?role=builder>)** Post your base in <#1483455027250200639> or <#1483489266947461321>.\n" +
		"Use `/submit-guild` or `/submit-solo` — the bot sends you a ready-to-paste template via DM. Your build appears on next sync.\n" +
		"**[Explorer?](<https://www.wherebuildersmeet.com/contribute/?role=scout>)** Use `/scout-guild` to report impressive bases you've found in <#1490051558237405254>.\n" +
		"**[Voter?](<https://www.wherebuildersmeet.com/contribute/?role=voter>)** React to showcase threads: ⭐ = 2 pts, 👍 🔥 = 1 pt each. Votes shape the public rankings.\n" +
		"Questions? Ask in the chats or ping a moderator. Happy Building!"
}

func handleWelcomeTestCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.Member.User.GlobalName
	if name == "" {
		name = i.Member.User.Username
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: BuildWelcomeMessage(name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("responding to welcome-test", "err", err)
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
