package discord

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

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
