package discord

import (
	"fmt"
	"strings"
)

func FormatCombinedSyncSummary(guildStats, soloStats SyncStats, hasSolo bool) string {
	line1 := fmt.Sprintf("🏯 **%d** guilds", guildStats.Total)
	if hasSolo {
		line1 += fmt.Sprintf(" · 🏡 **%d** solos", soloStats.Total)
	}
	line1 += " — synced!"

	var activity []string

	var updatedNames []string
	updatedNames = append(updatedNames, guildStats.UpdatedNames...)
	updatedNames = append(updatedNames, soloStats.UpdatedNames...)
	if len(updatedNames) > 0 {
		activity = append(activity, "🔄 "+strings.Join(updatedNames, ", "))
	}

	var newLinks []string
	for _, name := range guildStats.NewNames {
		if link, ok := guildStats.NewThreadLinks[name]; ok {
			newLinks = append(newLinks, fmt.Sprintf("[%s](%s)", name, link))
		} else {
			newLinks = append(newLinks, name)
		}
	}
	for _, name := range soloStats.NewNames {
		if link, ok := soloStats.NewThreadLinks[name]; ok {
			newLinks = append(newLinks, fmt.Sprintf("[%s](%s)", name, link))
		} else {
			newLinks = append(newLinks, name)
		}
	}
	if len(newLinks) > 0 {
		activity = append(activity, "🆕 "+strings.Join(newLinks, ", "))
	}

	if len(activity) > 0 {
		return line1 + "\n" + strings.Join(activity, " · ")
	}
	return line1
}
