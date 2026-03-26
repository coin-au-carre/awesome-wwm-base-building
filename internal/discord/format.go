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
	if s.New > 0 {
		lines = append(lines, fmt.Sprintf("🆕 **%d** new guild(s) discovered: %s",
			s.New, strings.Join(s.NewNames, ", ")))
	}
	if s.Updated > 0 {
		lines = append(lines, fmt.Sprintf("🔄 **%d** guild(s) refreshed: %s",
			s.Updated, strings.Join(s.UpdatedNames, ", ")))
	}
	return strings.Join(lines, "\n")
}
