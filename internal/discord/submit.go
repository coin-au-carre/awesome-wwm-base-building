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

}

func OnInteractionCreate(bot *Bot, root, submissionChannelID, discoveriesChannelID, guildForumChannelID string) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case submitCommandName:
				handleSubmitCommand(s, i)
			case postCommandName:
				handlePostCommand(s, i)
			}
		case discordgo.InteractionModalSubmit:
			switch i.ModalSubmitData().CustomID {
			case submitModalID:
				handleSubmitModal(s, i, bot, root, submissionChannelID, discoveriesChannelID)
			case postModalID:
				handlePostModal(s, i, guildForumChannelID)
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
						Label:       "Guild Name  —  ID optional",
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
		if i.Member.Nick != "" {
			submitter = i.Member.Nick
		} else if i.Member.User != nil {
			submitter = i.Member.User.Username
		}
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
						Label:       "Guild Name",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: "Iron Vanguard",
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "guild_id",
						Label:       "Guild ID — optional (8-digit number in-game)",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: "12345678  (Menu → Guild → Info)",
						MaxLength:   20,
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

func handlePostModal(s *discordgo.Session, i *discordgo.InteractionCreate, guildForumChannelID string) {
	fields := modalFields(i.ModalSubmitData().Components)
	name := fields["name"]
	guildID := strings.TrimSpace(fields["guild_id"])
	builders := fields["builders"]
	lore := fields["lore"]
	whatToVisit := fields["what_to_visit"]

	if guildForumChannelID == "" {
		slog.Error("submit-base: GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID not set")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Forum channel not configured. Please contact a moderator.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Build thread title: "GuildName [ID]" or just "GuildName"
	threadTitle := name
	if guildID != "" {
		threadTitle = fmt.Sprintf("%s [%s]", name, guildID)
	}

	// Build first post matching the website template format
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

	thread, err := s.ForumThreadStartComplex(guildForumChannelID, &discordgo.ThreadStart{
		Name:                threadTitle,
		AutoArchiveDuration: 10080, // 7 days
	}, &discordgo.MessageSend{
		Content: strings.TrimSpace(content.String()),
	})
	if err != nil {
		slog.Error("creating forum thread", "name", name, "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Something went wrong creating your thread. Please try again or post manually.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_, _ = s.ChannelMessageSend(thread.ID, "📸 Drop your screenshots here! The more the better 👇")

	slog.Info("submit-base thread created", "user", i.Member.User.Username, "name", name, "thread", thread.ID)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Your thread for **%s** has been created! Go drop your screenshots in <#%s> 📸", name, thread.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
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
