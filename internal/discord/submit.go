package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
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

	guildLinkCommandName = "guild"
	soloLinkCommandName  = "solo"
	randomCommandName    = "random"
	myVotesCommandName   = "my-votes"
	helpCommandName      = "commands"
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

func memberDisplayName(i *discordgo.InteractionCreate) string {
	if i.Member == nil || i.Member.User == nil {
		return "unknown"
	}
	if i.Member.Nick != "" {
		return i.Member.Nick
	}
	if i.Member.User.GlobalName != "" {
		return i.Member.User.GlobalName
	}
	return i.Member.User.Username
}

func RegisterSubmitCommand(s *discordgo.Session, discordGuildID string) {
	adminPerm := int64(discordgo.PermissionAdministrator)
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, discordGuildID, []*discordgo.ApplicationCommand{
		{
			Name:        submitCommandName,
			Description: "Scout a guild base and reference it for later",
		},
		{
			Name:        postCommandName,
			Description: "Submit your guild base to showcase",
		},
		{
			Name:        soloCommandName,
			Description: "Submit your solo construction to showcase",
		},
		{
			Name:                     welcomeTestCommandName,
			Description:              "Preview the welcome message (admins only)",
			DefaultMemberPermissions: &adminPerm,
		},
		{
			Name:        guildLinkCommandName,
			Description: "Share a guild base with its thread and page",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "name",
					Description:  "Guild name",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        soloLinkCommandName,
			Description: "Share a solo base with its thread and page",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "name",
					Description:  "Solo build name",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        randomCommandName,
			Description: "Ruby picks a random guild base to showcase",
		},
		{
			Name:        myVotesCommandName,
			Description: "Show the guild bases you have voted for (only you see the result)",
		},
		{
			Name:        helpCommandName,
			Description: "List all Ruby commands (only you see the result)",
		},
	})
	if err != nil {
		slog.Error("registering commands", "err", err)
	}
}

func OnInteractionCreate(bot *Bot, root, submissionChannelID, discoveriesChannelID, guildForumChannelID, soloForumChannelID string, responder *Responder) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommandAutocomplete:
			switch i.ApplicationCommandData().Name {
			case guildLinkCommandName:
				handleGuildLinkAutocomplete(s, i, root)
			case soloLinkCommandName:
				handleSoloLinkAutocomplete(s, i, root)
			}
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
			case guildLinkCommandName:
				handleGuildLinkCommand(s, i, root)
			case soloLinkCommandName:
				handleSoloLinkCommand(s, i, root)
			case randomCommandName:
				handleRandomCommand(s, i, root, responder)
			case myVotesCommandName:
				handleMyVotesCommand(s, i, root)
			case helpCommandName:
				handleHelpCommand(s, i)
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
	slog.Info("scout-guild command received", "user", memberDisplayName(i))
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
						CustomID:    "builders_proposed",
						Label:       "Builders (optional)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "Builder1, Builder2",
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
	builders := splitCSV(fields["builders_proposed"])

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

	scoutedByID := ""
	if i.Member != nil && i.Member.User != nil {
		scoutedByID = i.Member.User.ID
	}

	g := guild.Guild{
		ID:                 guildID,
		Name:               name,
		WhatToVisit:        whatToVisit,
		Builders:           builders,
		Tags:               tags,
		Score:              score,
		LastModified:       guild.ModifiedNow(),
		ScoutedByDiscordID: scoutedByID,
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

	slog.Info("guild proposed", "user", memberDisplayName(i), "name", name, "appreciation", appreciation, "score", score)

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
	slog.Info("submit-guild command received", "user", memberDisplayName(i))
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

	slog.Info("submit-guild form received", "user", memberDisplayName(i), "name", name)

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
	slog.Info("submit-solo command received", "user", memberDisplayName(i))
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

	slog.Info("submit-solo form received", "user", memberDisplayName(i), "name", name)

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
	return "**Welcome to [Where Builders Meet](<https://www.wherebuildersmeet.com?utm_source=discord&utm_medium=welcome>), " + name + "!** :wave:\n\n" +
		"**New to building?** Browse [build inspiration & media](<https://www.wherebuildersmeet.com/media/?utm_source=discord&utm_medium=welcome>) to get ideas, and check out <#1483483683456286911> for construction help and <#1483447711499030634> for tips and tricks.\n" +
		"**[Builder?](<https://www.wherebuildersmeet.com/contribute/?role=builder&utm_source=discord&utm_medium=welcome>)** Post your base in <#1483455027250200639> or <#1483489266947461321>.\n" +
		"Use `/submit-guild` or `/submit-solo` — the bot sends you a ready-to-paste template via DM. Your build appears on next sync.\n" +
		"**[Explorer?](<https://www.wherebuildersmeet.com/contribute/?role=scout&utm_source=discord&utm_medium=welcome>)** Use `/scout-guild` to report impressive bases you've found in <#1490051558237405254>.\n" +
		"**[Voter?](<https://www.wherebuildersmeet.com/contribute/?role=voter&utm_source=discord&utm_medium=welcome>)** React to showcase threads: ⭐ = 2 pts, 👍 🔥 = 1 pt each. Votes shape the public rankings.\n" +
		"Questions? Ask in the chats or ping a moderator. Happy Building!"
}

func handleWelcomeTestCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := memberDisplayName(i)
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

const maxAutocompleteChoices = 25

func handleGuildLinkAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	query := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			query = strings.ToLower(strings.TrimSpace(opt.StringValue()))
			break
		}
	}

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Warn("loading guilds for autocomplete", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{},
		})
		return
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, g := range guilds {
		if query == "" || strings.Contains(strings.ToLower(g.Name), query) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  g.Name,
				Value: g.Name,
			})
			if len(choices) >= maxAutocompleteChoices {
				break
			}
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func handleGuildLinkCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	query := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			query = opt.StringValue()
			break
		}
	}

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Error("loading guilds for guild link", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't load guild data, try again later)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	lower := strings.ToLower(query)
	var match *guild.Guild
	for idx := range guilds {
		if strings.ToLower(guilds[idx].Name) == lower {
			match = &guilds[idx]
			break
		}
	}
	if match == nil {
		for idx := range guilds {
			if strings.Contains(strings.ToLower(guilds[idx].Name), lower) {
				match = &guilds[idx]
				break
			}
		}
	}

	if match == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("*(no guild found matching \"%s\")*", query),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: guildLinkContent(match),
		},
	})
}

func handleRandomCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string, _ *Responder) {
	guilds, err := guild.Load(root)
	if err != nil {
		slog.Error("loading guilds for random", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't find the guilds scroll... something went wrong!)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	pick, _, ok := PickRandomGuild(guilds)
	if !ok {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(no guild bases with screenshots yet... come back soon!)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	slog.Info("random slash command sent", "guild", pick.Name)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: FormatSpotlightMessage(pick, true),
		},
	})
}

func soloLinkContent(g *guild.Guild) string {
	siteURL := websiteURL("/solos/"+slugify(g.Name), "solo_cmd")
	if g.DiscordThread != "" {
		return fmt.Sprintf("**%s** · %s · [WBM page](%s)", g.Name, g.DiscordThread, siteURL)
	}
	return fmt.Sprintf("**%s** · [WBM page](%s)", g.Name, siteURL)
}

func loadSolos(root string) ([]guild.Guild, error) {
	return guild.LoadFile(root + "/data/solos.json")
}

func handleSoloLinkAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	query := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			query = strings.ToLower(strings.TrimSpace(opt.StringValue()))
			break
		}
	}

	solos, err := loadSolos(root)
	if err != nil {
		slog.Warn("loading solos for autocomplete", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{},
		})
		return
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, g := range solos {
		if query == "" || strings.Contains(strings.ToLower(g.Name), query) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  g.Name,
				Value: g.Name,
			})
			if len(choices) >= maxAutocompleteChoices {
				break
			}
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func handleSoloLinkCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	query := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "name" {
			query = opt.StringValue()
			break
		}
	}

	solos, err := loadSolos(root)
	if err != nil {
		slog.Error("loading solos for solo link", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't load solo data, try again later)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	lower := strings.ToLower(query)
	var match *guild.Guild
	for idx := range solos {
		if strings.ToLower(solos[idx].Name) == lower {
			match = &solos[idx]
			break
		}
	}
	if match == nil {
		for idx := range solos {
			if strings.Contains(strings.ToLower(solos[idx].Name), lower) {
				match = &solos[idx]
				break
			}
		}
	}

	if match == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("*(no solo build found matching \"%s\")*", query),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: soloLinkContent(match),
		},
	})
}

func guildLinkContent(g *guild.Guild) string {
	siteURL := websiteURL("/guilds/"+slugify(g.Name), "guild_cmd")
	if g.DiscordThread != "" {
		return fmt.Sprintf("**%s** · %s · [WBM page](%s)", g.Name, g.DiscordThread, siteURL)
	}
	return fmt.Sprintf("**%s** · [WBM page](%s)", g.Name, siteURL)
}

var emojiDisplayOrder = []string{"⭐", "👍", "🔥", "❤️"}

func normalizeEmoji(emoji string) string {
	switch emoji {
	case "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿":
		return "👍"
	}
	return emoji
}

func reactionPoints(emoji string) int {
	switch emoji {
	case "⭐":
		return scorePerStar
	case "👍", "👍🏻", "👍🏼", "👍🏽", "👍🏾", "👍🏿", "🔥", "❤️":
		return scorePerLike
	}
	return 0
}

// userReactionsForID returns the display string and total raw pts for a user ID in a per-thread emoji map.
func userReactionsForID(emojiMap map[string][]string, userID string) (string, int) {
	found := map[string]bool{}
	for emoji, ids := range emojiMap {
		for _, uid := range ids {
			if uid == userID {
				found[normalizeEmoji(emoji)] = true
				break
			}
		}
	}
	if len(found) == 0 {
		return "", 0
	}
	pts := 0
	var display []string
	for _, e := range emojiDisplayOrder {
		if found[e] {
			display = append(display, e)
			pts += reactionPoints(e)
		}
	}
	return strings.Join(display, " "), pts
}

func threadIDFromURL(u string) string {
	i := strings.LastIndex(u, "/")
	if i < 0 {
		return u
	}
	return u[i+1:]
}

func handleMyVotesCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	if i.Member == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(this command only works in a server)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	userID := i.Member.User.ID

	reactions, _ := guild.LoadReactions(root)

	// Build threadID → entry name from both guilds and solos.
	threadToName := make(map[string]string)
	if guilds, err := guild.Load(root); err == nil {
		for _, g := range guilds {
			if tid := threadIDFromURL(g.DiscordThread); tid != "" {
				threadToName[tid] = g.Name
			}
		}
	}
	if solos, err := loadSolos(root); err == nil {
		for _, g := range solos {
			if tid := threadIDFromURL(g.DiscordThread); tid != "" {
				threadToName[tid] = g.Name
			}
		}
	}

	type entry struct {
		name   string
		emojis string
		pts    int
	}

	var entries []entry
	for threadID, emojiMap := range reactions {
		name, ok := threadToName[threadID]
		if !ok {
			continue
		}
		if emojis, pts := userReactionsForID(emojiMap, userID); emojis != "" {
			entries = append(entries, entry{name, emojis, pts})
		}
	}

	if len(entries) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(no votes found for your account)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sort.Slice(entries, func(a, b int) bool {
		if entries[a].pts != entries[b].pts {
			return entries[a].pts > entries[b].pts
		}
		return entries[a].name < entries[b].name
	})

	totalPts := 0
	for _, e := range entries {
		totalPts += e.pts
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "**Your votes** · %d guilds · %d pts\n\n", len(entries), totalPts)

	rank := 0
	for idx, e := range entries {
		if idx == 0 || entries[idx-1].pts != e.pts {
			rank = idx + 1
		}
		fmt.Fprintf(&sb, "**#%d** %s · +%d %s\n", rank, e.name, e.pts, e.emojis)
		if sb.Len() > 1800 {
			sb.WriteString("*... and more*")
			break
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: strings.TrimRight(sb.String(), "\n"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

const helpMessage = `**Ruby's commands**

**Discover**
/random — Ruby picks a random guild base to showcase
/guild \<name\> — share a guild base with its thread and page
/solo \<name\> — share a solo base with its thread and page

**Contribute**
/scout-guild — scout a guild base and reference it for later
/submit-guild — submit your guild base to showcase
/submit-solo — submit your solo construction the showcase

**Personal**
/my-votes — show the guild bases you have voted for (only you see the result)
/commands — show this list (only you see the result)`

func handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
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
