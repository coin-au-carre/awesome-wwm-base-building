package discord

import (
	"fmt"
	"strings"
)

func FormatSyncSummary(s SyncStats) string {
	lines := []string{
		"✨ **All guilds have been synchronized!**",
		fmt.Sprintf("🏰 **%d** guilds tracked", s.Total),
	}
	if s.Updated > 0 {
		lines = append(lines, fmt.Sprintf("🔄 **%d** guild(s) refreshed: %s",
			s.Updated, strings.Join(s.UpdatedNames, ", ")))
	}
	if s.New > 0 {
		lines = append(lines, fmt.Sprintf("🆕 **%d** new guild(s) discovered:", s.New))
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
