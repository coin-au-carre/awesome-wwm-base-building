// cmd/check-abuse/main.go — offline voter abuse detection from saved reaction data
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	root          := flag.String("root", cmdutil.RootDir(), "root directory containing data/")
	minThreads    := flag.Int("min-threads", 4, "minimum guilds voted on before a voter can be flagged (matches weight ×1 threshold)")
	minOtherHigh  := flag.Int("min-other-high", 1, "min other guilds that must also have ≥4 pts for the voter to be considered legitimate")
	hide          := flag.Bool("hide", false, "redact user IDs and guild names from output (safe to share publicly)")
	flag.Parse()

	reactions, err := guild.LoadReactions(*root)
	if err != nil {
		slog.Error("loading reactions", "err", err)
		os.Exit(1)
	}
	if len(reactions) == 0 {
		slog.Error("reactions.json is empty or missing — run task sync first")
		os.Exit(1)
	}

	users, err := guild.LoadUsers(*root)
	if err != nil {
		slog.Warn("could not load users.json, IDs will be shown instead", "err", err)
	}

	blacklist, err := guild.LoadVoterBlacklist(filepath.Join(*root, "data", "voter_blacklist.json"))
	if err != nil {
		slog.Warn("could not load voter_blacklist.json", "err", err)
	}

	whitelist, err := guild.LoadVoterBlacklist(filepath.Join(*root, "data", "voter_whitelist.json"))
	if err != nil {
		slog.Warn("could not load voter_whitelist.json", "err", err)
	}

	guilds, _ := guild.LoadFile(filepath.Join(*root, "data", "guilds.json"))
	guildNameByThreadID := buildNameMap(guilds)

	// Keep only threads that belong to guild showcases.
	guildThreadIDs := buildThreadIDSet(guilds)
	for tid := range reactions {
		if !guildThreadIDs[tid] {
			delete(reactions, tid)
		}
	}

	all := discord.SummarizeVoters(reactions, guildNameByThreadID)

	var suspects []discord.VoterStats
	for _, s := range all {
		if whitelist[s.UserID] {
			continue
		}
		if s.Threads >= *minThreads &&
			s.TopRawPts >= 4 &&
			s.HighScoreOthers < *minOtherHigh {
			suspects = append(suspects, s)
		}
	}
	sort.Slice(suspects, func(i, j int) bool {
		if suspects[i].HighScoreOthers != suspects[j].HighScoreOthers {
			return suspects[i].HighScoreOthers < suspects[j].HighScoreOthers
		}
		return suspects[i].TopRawPts > suspects[j].TopRawPts
	})

	fmt.Printf("=== Suspicious voters (%d) — ≥4 pts to top guild, <%d other guild(s) with ≥4 pts, min %d guilds voted ===\n",
		len(suspects), *minOtherHigh, *minThreads)
	if len(suspects) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, s := range suspects {
		blacklisted := ""
		if blacklist[s.UserID] {
			blacklisted = "  [blacklisted]"
		}
		avgOther := 0.0
		if s.Threads > 1 {
			avgOther = float64(s.TotalRawPts-s.TopRawPts) / float64(s.Threads-1)
		}
		name := displayName(s.UserID, users)
		guildName := s.TopGuildName
		uid := s.UserID
		if *hide {
			name = "[hidden]"
			uid = "[hidden]"
		}
		fmt.Printf("  %-30s  top: %-30s  %dpts → capped %dpts  (avg others: %.1f pts, %d guilds)  ID: %s%s\n",
			name,
			guildName,
			s.TopRawPts,
			s.Cap,
			avgOther,
			s.Threads,
			uid,
			blacklisted,
		)
	}

	fmt.Println("\nTo blacklist a voter, add their ID to data/voter_blacklist.json.")
}

func displayName(uid string, users guild.UserMap) string {
	if info, ok := users[uid]; ok {
		return info.DisplayName()
	}
	return uid
}

func buildThreadIDSet(all []guild.Guild) map[string]bool {
	m := make(map[string]bool, len(all))
	for _, g := range all {
		if g.DiscordThread == "" {
			continue
		}
		parts := strings.Split(g.DiscordThread, "/")
		if len(parts) > 0 {
			m[parts[len(parts)-1]] = true
		}
	}
	return m
}

// buildNameMap extracts threadID → guild name from the discordThread URL field.
func buildNameMap(all []guild.Guild) map[string]string {
	m := make(map[string]string, len(all))
	for _, g := range all {
		if g.DiscordThread == "" {
			continue
		}
		parts := strings.Split(g.DiscordThread, "/")
		if len(parts) > 0 {
			m[parts[len(parts)-1]] = g.Name
		}
	}
	return m
}
