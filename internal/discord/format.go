package discord

import (
	"fmt"
	"strings"
)

func FormatSyncSummary(s SyncStats) string {
	return formatSyncSummary(s, false)
}

func FormatSoloSyncSummary(s SyncStats) string {
	return formatSyncSummary(s, true)
}

func formatSyncSummary(s SyncStats, isSolo bool) string {
	var lines []string
	if isSolo {
		lines = []string{
			fmt.Sprintf("🏡 **%d** solo construction tracked & synced!", s.Total),
		}
	} else {
		lines = []string{
			fmt.Sprintf("🏯 **%d** guilds tracked & synced!", s.Total),
		}
	}
	kind := "guild"
	if isSolo {
		kind = "solo build"
	}
	if s.Updated > 0 {
		lines = append(lines, fmt.Sprintf("🔄 **%d** %s(s) refreshed: %s",
			s.Updated, kind, strings.Join(s.UpdatedNames, ", ")))
	}
	if s.New > 0 {
		lines = append(lines, fmt.Sprintf("🆕 **%d** new %s(s) discovered:", s.New, kind))
		for _, name := range s.NewNames {
			if link, ok := s.NewThreadLinks[name]; ok {
				lines = append(lines, fmt.Sprintf("  • [%s](%s)", name, link))
			} else {
				lines = append(lines, fmt.Sprintf("  • %s", name))
			}
		}
	}
	return strings.Join(lines, "\n")
}
