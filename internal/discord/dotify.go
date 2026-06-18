package discord

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleDotifyCommand replaces dots in a URL with ｡ (U+FF61) so it can be
// shared in Where Builders Meet without triggering the anti-URL filter.
func handleDotifyCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	url := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "url" {
			url = strings.TrimSpace(opt.StringValue())
			break
		}
	}

	result := strings.ReplaceAll(url, ".", "｡")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "```\n" + result + "\n```\n<" + url + ">",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
