package discord

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const helpMessage = `**Ruby's commands**

**Discover**
/random — Ruby picks a random guild base to showcase
/guild \<name\> — share a guild base with its thread and page
/solo \<name\> — share a solo base with its thread and page
/builder \<member\> — look up a member's bases (only you see the result)

**Contribute**
/scout-guild — scout a guild base and reference it for later
/submit-guild — submit your guild base to showcase
/submit-solo — submit your solo construction the showcase

**Personal**
/my-votes — show the guild bases you have voted for (only you see the result)
/commands — show this list (only you see the result)`

func handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var serverName string
	if g, err := s.Guild(i.GuildID); err == nil {
		serverName = g.Name
	} else {
		serverName = i.GuildID
	}
	slog.Info("/commands command used", "user", memberDisplayName(i), "server", serverName)

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpMessage,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleBuilderCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	var targetUser *discordgo.User
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "member" {
			targetUser = opt.UserValue(s)
			break
		}
	}

	var serverName string
	if g, err := s.Guild(i.GuildID); err == nil {
		serverName = g.Name
	} else {
		serverName = i.GuildID
	}
	if targetUser != nil {
		slog.Info("/builder command used", "user", memberDisplayName(i), "server", serverName, "target", targetUser.Username)
	}

	if targetUser == nil {
		slog.Error("couldn't resolve the selected member")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't resolve the selected member)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	userID := targetUser.ID

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Error("loading guilds for builder command", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't load guild data, try again later)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	solos, err := loadSolos(root)
	if err != nil {
		slog.Error("loading solos for builder command", "err", err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't load solo data, try again later)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	type baseEntry struct {
		g      guild.Guild
		isSolo bool
	}
	var matches []baseEntry
	for _, g := range guilds {
		if g.PosterDiscordID == userID {
			if !slices.Contains([]string{AHLYAM_ID, BABE_ID, WINDXP_ID}, g.PosterDiscordID) ||
				g.Name == "Jenova" || g.GuildName == "PleasureSeeker" {
				matches = append(matches, baseEntry{g, false})
			}
		}
	}
	for _, g := range solos {
		if g.PosterDiscordID == userID {
			matches = append(matches, baseEntry{g, true})
		}
	}

	if len(matches) == 0 {
		displayName := targetUser.GlobalName
		if displayName == "" {
			displayName = targetUser.Username
		}
		resolved := i.ApplicationCommandData().Resolved
		if resolved != nil {
			if m, ok := resolved.Members[userID]; ok && m.Nick != "" {
				displayName = m.Nick
			}
		}
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("*(no guild or solo base found for **%s**)*", displayName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var sb strings.Builder
	var buttons []discordgo.MessageComponent
	for idx, m := range matches {
		if len(buttons) >= 5 {
			if idx == 5 {
				s := "s"
				if len(matches)-5 == 1 {
					s = ""
				}
				fmt.Fprintf(&sb, "*(and %d more base%s)*", len(matches)-5, s)
			}
			break
		}
		if m.isSolo {
			sb.WriteString(soloLinkContent(&m.g) + "\n")
		} else {
			sb.WriteString(guildLinkContent(&m.g) + "\n")
		}
		prefix := shareGuildPrefix
		if m.isSolo {
			prefix = shareSoloPrefix
		}
		customID := prefix + m.g.Name
		if len(customID) > 100 {
			customID = customID[:100]
		}
		label := "Share " + m.g.Name
		if len(label) > 80 {
			label = label[:80]
		}
		buttons = append(buttons, discordgo.Button{
			Style:    discordgo.PrimaryButton,
			Label:    label,
			CustomID: customID,
		})
	}

	var components []discordgo.MessageComponent
	if len(buttons) > 0 {
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: buttons},
		}
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    strings.TrimRight(sb.String(), "\n"),
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	})
}
