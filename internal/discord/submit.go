package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

var knownTags = map[string]bool{
	"Arena": true, "Cave": true, "City": true, "Creative": true,
	"Cute": true, "Dance Floor": true, "Desert": true, "Floating Island": true,
	"Fun": true, "Maze": true, "Military": true, "Mountain": true,
	"Nature": true, "River": true, "Snow": true, "Zen": true,
}

var appreciationScore = map[string]int{
	"s": 2, "S": 2,
	"a": 1, "A": 1,
	"b": 0, "B": 0,
}

func handleSubmitCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var guildName string
	if guild, err := s.Guild(i.GuildID); err == nil {
		guildName = guild.Name
	} else {
		guildName = i.GuildID
	}
	slog.Info("/scout-guild command used", "user", memberDisplayName(i), "server", guildName)
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

func buildDiscoveryMessage(explorer string, g guild.Guild) string {
	title := g.Name
	if g.ID != "" {
		title = fmt.Sprintf("%s [%s]", g.Name, g.ID)
	}

	builder := ""
	if len(g.Builders) > 0 {
		builder = strings.Join(g.Builders, ", ")
	}

	line1 := fmt.Sprintf("🧭 **%s** by %s — *scouted by %s*", title, builder, explorer)
	if builder == "" {
		line1 = fmt.Sprintf("🧭 **%s** — *scouted by %s*", title, explorer)
	}

	suffix := "||"

	const (
		discordMax = 2000
		ellipsis   = " [...]"
	)
	// fixed parts: line1 + "\n" + "||" + wtv + suffix
	fixed := len(line1) + 1 + 2 + len(suffix)
	wtv := g.WhatToVisit
	if fixed+len(wtv) > discordMax {
		wtv = wtv[:discordMax-fixed-len(ellipsis)] + ellipsis
	}

	return line1 + "\n||" + wtv + suffix
}

func handlePostCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var guildName string
	if guild, err := s.Guild(i.GuildID); err == nil {
		guildName = guild.Name
	} else {
		guildName = i.GuildID
	}
	slog.Info("/submit-guild command used", "user", memberDisplayName(i), "server", guildName)
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

	threadTitle := name
	if guildID != "" {
		threadTitle = fmt.Sprintf("%s [%s]", name, guildID)
	}

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
	var guildName string
	if guild, err := s.Guild(i.GuildID); err == nil {
		guildName = guild.Name
	} else {
		guildName = i.GuildID
	}
	slog.Info("/submit-solo command used", "user", memberDisplayName(i), "server", guildName)
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

// BuildWelcomeMessage returns the welcome message for a new member.
func BuildWelcomeMessage(name string) string {
	return "**Welcome to [Where Builders Meet](<https://www.wherebuildersmeet.com?utm_source=discord&utm_medium=welcome>), " + name + "!** :wave:\n\n" +
		"**New to building?** Browse [tutorials & videos](<https://www.wherebuildersmeet.com/media/?utm_source=discord&utm_medium=welcome>), check out <#1483483683456286911> to ask questions and <#1483447711499030634> for ideas.\n" +
		"**[Want to vote?](<https://www.wherebuildersmeet.com/contribute/voter?utm_source=discord&utm_medium=welcome>)** Read the [voting guide](<https://www.wherebuildersmeet.com/how-it-works/?utm_source=discord&utm_medium=welcome>) before you start. Fair voting keeps the rankings honest and this project alive — explore many guilds and rate each one honestly. Your votes carry more weight the more builds you've visited.\n" +
		"**Builder?** Check the [Contribute page](<https://www.wherebuildersmeet.com/contribute/builder?utm_source=discord&utm_medium=welcome>) to see your construction on the website and in <#1483455027250200639> or <#1483489266947461321>.\n" +
		"\tUse `/submit-guild` or `/submit-solo` — the bot sends you a ready-to-paste template via DM. Your build appears on next sync.\n" +
		"**[Explorer?](<https://www.wherebuildersmeet.com/contribute/scout?utm_source=discord&utm_medium=welcome>)** Use `/scout-guild` to report impressive bases you've found in <#1490051558237405254>.\n" +
		"Looking for a builder help? ask in <#1486701728551407796>\n" +
		"Questions? Ask in the chats, <#1499187607790157996> or ping a moderator. Happy Building!"
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
