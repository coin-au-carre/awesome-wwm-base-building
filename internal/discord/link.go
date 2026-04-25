package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

const maxAutocompleteChoices = 25

func loadSolos(root string) ([]guild.Guild, error) {
	return guild.LoadFile(root + "/data/solos.json")
}

func guildLinkContent(g *guild.Guild) string {
	siteURL := websiteURL("/guilds/"+slugify(g.Name), "guild_cmd")
	if g.DiscordThread != "" {
		return fmt.Sprintf("**%s** · %s · [WBM page](%s)", g.Name, g.DiscordThread, siteURL)
	}
	return fmt.Sprintf("**%s** · [WBM page](%s)", g.Name, siteURL)
}

func soloLinkContent(g *guild.Guild) string {
	siteURL := websiteURL("/solos/"+slugify(g.Name), "solo_cmd")
	if g.DiscordThread != "" {
		return fmt.Sprintf("**%s** · %s · [WBM page](%s)", g.Name, g.DiscordThread, siteURL)
	}
	return fmt.Sprintf("**%s** · [WBM page](%s)", g.Name, siteURL)
}

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

	var serverName string
	if g, err := s.Guild(i.GuildID); err == nil {
		serverName = g.Name
	} else {
		serverName = i.GuildID
	}
	slog.Info("/guild command used", "user", memberDisplayName(i), "server", serverName, "query", query)

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

	match := findEntry(guilds, query)
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

	var serverName string
	if g, err := s.Guild(i.GuildID); err == nil {
		serverName = g.Name
	} else {
		serverName = i.GuildID
	}
	slog.Info("/solo command used", "user", memberDisplayName(i), "server", serverName, "query", query)

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

	match := findEntry(solos, query)
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

func handleRandomCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root string, _ LLMResponder) {
	var serverName string
	if g, err := s.Guild(i.GuildID); err == nil {
		serverName = g.Name
	} else {
		serverName = i.GuildID
	}
	slog.Info("/random command used", "user", memberDisplayName(i), "server", serverName)

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
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: FormatSpotlightMessage(pick, true),
		},
	})
}

func handleShareBuilderButton(s *discordgo.Session, i *discordgo.InteractionCreate, root string) {
	customID := i.MessageComponentData().CustomID

	var isSolo bool
	var name string
	switch {
	case strings.HasPrefix(customID, shareGuildPrefix):
		name = customID[len(shareGuildPrefix):]
	case strings.HasPrefix(customID, shareSoloPrefix):
		name = customID[len(shareSoloPrefix):]
		isSolo = true
	default:
		return
	}

	var content string
	if isSolo {
		solos, err := loadSolos(root)
		if err == nil {
			if match := findEntry(solos, name); match != nil {
				content = soloLinkContent(match)
			}
		}
	} else {
		guilds, err := guild.Load(root)
		if err == nil {
			if match := findEntry(guilds, name); match != nil {
				content = guildLinkContent(match)
			}
		}
	}

	if content == "" {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "*(couldn't find the base, try again)*",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

// findEntry returns the first guild whose name exactly matches (case-insensitive),
// then falls back to the first that contains the query.
func findEntry(entries []guild.Guild, query string) *guild.Guild {
	lower := strings.ToLower(query)
	for idx := range entries {
		if strings.ToLower(entries[idx].Name) == lower {
			return &entries[idx]
		}
	}
	for idx := range entries {
		if strings.Contains(strings.ToLower(entries[idx].Name), lower) {
			return &entries[idx]
		}
	}
	return nil
}
