package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const (
	submitCommandName = "scout-guild"
	submitModalID     = "scout_guild_modal"
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
		slog.Error("registering propose-guild command", "err", err)
	}
}

func OnInteractionCreate(bot *Bot, root, submissionChannelID string) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if i.ApplicationCommandData().Name == submitCommandName {
				handleSubmitCommand(s, i)
			}
		case discordgo.InteractionModalSubmit:
			if i.ModalSubmitData().CustomID == submitModalID {
				handleSubmitModal(s, i, bot, root, submissionChannelID)
			}
		}
	}
}

func handleSubmitCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

func handleSubmitModal(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, root, submissionChannelID string) {
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
		LastModified: time.Now().UTC().Format("2006-01-02"),
	}
	if buildersProposed {
		n := guild.Note{Text: "Builders proposed to WBM."}
		g.Note = &n
	}

	submitMu.Lock()
	guilds, err := guild.Load(root)
	if err == nil {
		guilds = append(guilds, g)
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

	slog.Info("guild proposed", "name", name, "appreciation", appreciation, "score", score)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Guild **%s** scouted successfully! (appreciation: %s)", name, appreciation),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	submitter := "unknown"
	if i.Member != nil && i.Member.User != nil {
		submitter = i.Member.User.Username
	}

	jsonBytes, _ := json.MarshalIndent(g, "", "\t")
	notice := fmt.Sprintf("**New guild scouted** by %s\n```json\n%s\n```", submitter, string(jsonBytes))
	if submissionChannelID != "" {
		bot.Send(submissionChannelID, notice)
	} else {
		bot.Notify(notice)
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
