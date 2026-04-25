package discord

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

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

	var serverName string
	if g, err := s.Guild(i.GuildID); err == nil {
		serverName = g.Name
	} else {
		serverName = i.GuildID
	}
	slog.Info("/my-votes command used", "user", memberDisplayName(i), "server", serverName)

	userID := i.Member.User.ID
	reactions, _ := guild.LoadReactions(root)

	threadToName := make(map[string]string)
	var guilds []guild.Guild
	if loadedGuilds, err := guild.Load(root); err == nil {
		guilds = loadedGuilds
		for _, g := range guilds {
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

	currentPts := -1
	var currentNames []string
	var currentEmojis string

	for _, e := range entries {
		if e.pts != currentPts {
			if currentPts != -1 {
				fmt.Fprintf(&sb, "+%d pts [%s] %s\n", currentPts, currentEmojis, strings.Join(currentNames, ", "))
				if sb.Len() > 1800 {
					sb.WriteString("*... and more*")
					break
				}
			}
			currentPts = e.pts
			currentNames = []string{e.name}
			currentEmojis = e.emojis
		} else {
			currentNames = append(currentNames, e.name)
		}
	}

	if currentPts != -1 {
		fmt.Fprintf(&sb, "+%d pts [%s] %s\n", currentPts, currentEmojis, strings.Join(currentNames, ", "))
	}

	votedNames := make(map[string]bool)
	for _, e := range entries {
		votedNames[e.name] = true
	}

	var suggestions []string
	for _, g := range guilds {
		if !votedNames[g.Name] && len(g.Screenshots) > 0 && g.PosterDiscordID != AHLYAM_ID && g.PosterDiscordID != WINDXP_ID {
			suggestions = append(suggestions, g.Name)
		}
	}

	if len(suggestions) > 0 {
		fmt.Fprintf(&sb, "\n**Guild base suggestions to explore:**")
		i := 0
		for i < len(suggestions) && sb.Len() < 1800 {
			var line []string
			for i < len(suggestions) {
				testLine := append(line, suggestions[i])
				testText := "\n" + strings.Join(testLine, ", ")
				if len(sb.String())+len(testText) > 1800 && len(line) > 0 {
					break
				}
				line = append(line, suggestions[i])
				i++
			}
			if len(line) > 0 {
				fmt.Fprintf(&sb, "\n%s", strings.Join(line, ", "))
			}
		}
		if i < len(suggestions) {
			sb.WriteString("\n*... and more*")
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
