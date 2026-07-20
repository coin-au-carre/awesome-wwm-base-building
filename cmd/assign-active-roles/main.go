package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

// wandererRoleID is the "Wanderer" role, fixed and never changes.
const wandererRoleID = "1525608229613338634"

var trackedChannels = []string{
	"1483447711499030633", // chat-wwm
	"1521760524235309191", // homestead-system
	"1514776235299967017", // guild-building
	"1514776286260756480", // solo-building
	"1483483683456286911", // construction-help
	"1522001435187744901", // newbie-corner
	"1483447711499030634", // tips-and-tricks
	"1483451090048520252", // whatever-showcase
}

func main() {
	threshold := flag.Int("threshold", 100, "minimum lifetime messages across tracked channels to earn the Wanderer role")
	flag.Parse()

	root := cmdutil.RootDir()
	cmdutil.LoadEnv(root)

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	baseBuilderRoleID := cmdutil.RequireEnv("BASE_BUILDER_ROLE_ID")
	soloBuilderRoleID := os.Getenv("SOLO_BUILDER_ROLE_ID")
	if soloBuilderRoleID == "" {
		soloBuilderRoleID = baseBuilderRoleID
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating Discord session", "err", err)
		os.Exit(1)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	activityPath := filepath.Join(root, "data", "chat_activity.json")
	state, err := discord.LoadActivityState(activityPath)
	if err != nil {
		slog.Error("loading activity state", "err", err)
		os.Exit(1)
	}
	slog.Info("starting scan", "channels", len(trackedChannels), "threshold", *threshold)

	if err := discord.CollectActivity(session, trackedChannels, state); err != nil {
		slog.Error("collecting channel activity", "err", err)
		os.Exit(1)
	}

	if err := state.Save(activityPath); err != nil {
		slog.Error("saving activity state", "err", err)
		os.Exit(1)
	}
	slog.Info("scan complete", "known_users", len(state.Counts))

	ch, err := session.Channel(trackedChannels[0])
	if err != nil {
		slog.Error("fetching channel", "err", err)
		os.Exit(1)
	}
	discordGuildID := ch.GuildID

	users, err := guild.LoadUsers(root)
	if err != nil {
		slog.Error("loading discord_users.json", "err", err)
		os.Exit(1)
	}
	usersDirty := backfillUsers(session, discordGuildID, users)
	if usersDirty {
		if err := guild.SaveUsers(root, users); err != nil {
			slog.Error("saving discord_users.json", "err", err)
			os.Exit(1)
		}
	}

	roleCachePath := filepath.Join(root, "data", "role_assignments.json")
	roleCache, err := discord.LoadRoleCache(roleCachePath)
	if err != nil {
		slog.Error("loading role cache", "err", err)
		os.Exit(1)
	}

	assigned := 0
	for userID, count := range state.Counts {
		if count < *threshold {
			continue
		}
		name := displayName(users, userID)
		slog.Info("wanderer eligible", "user", name, "id", userID, "messages", count)
		discord.AssignAwesomeBuilderRole(session, discordGuildID, userID, wandererRoleID, name, roleCache)
		assigned++
	}
	if err := roleCache.Save(); err != nil {
		slog.Error("saving role cache", "err", err)
		os.Exit(1)
	}
	slog.Info("wanderer role assignment done", "eligible", assigned, "threshold", *threshold)

	reportPath := filepath.Join(root, "data", "inactive_builders.txt")
	if err := reportInactiveBuilders(reportPath, roleCachePath, baseBuilderRoleID, soloBuilderRoleID, state, users, *threshold); err != nil {
		slog.Error("writing inactive builders report", "err", err)
		os.Exit(1)
	}
}

// backfillUsers fetches every guild member once and stores identity info
// (username/globalName/nickname) for anyone missing or stale in discord_users.json.
func backfillUsers(session *discordgo.Session, discordGuildID string, users guild.UserMap) bool {
	var members []*discordgo.Member
	var after string
	for {
		page, err := session.GuildMembers(discordGuildID, after, 1000)
		if err != nil {
			slog.Warn("fetching guild members for nickname backfill", "err", err)
			return false
		}
		if len(page) == 0 {
			break
		}
		members = append(members, page...)
		after = page[len(page)-1].User.ID
		if len(page) < 1000 {
			break
		}
	}
	slog.Info("fetched guild members for nickname backfill", "count", len(members))

	dirty := false
	for _, m := range members {
		fresh := guild.UserInfo{
			Username:   m.User.Username,
			GlobalName: m.User.GlobalName,
			Nickname:   m.Nick,
		}
		if info, known := users[m.User.ID]; !known || info != fresh {
			users[m.User.ID] = fresh
			dirty = true
		}
	}
	return dirty
}

func displayName(users guild.UserMap, userID string) string {
	if info, ok := users[userID]; ok {
		return info.DisplayName()
	}
	return userID
}

// reportInactiveBuilders writes reportPath with builders who hold the guild
// or solo builder role but fall short of the chat threshold. No role change,
// read-only report.
func reportInactiveBuilders(reportPath, roleCachePath, baseBuilderRoleID, soloBuilderRoleID string, state *discord.ActivityState, users guild.UserMap, threshold int) error {
	data, err := os.ReadFile(roleCachePath)
	if err != nil {
		return fmt.Errorf("reading role assignments: %w", err)
	}
	var byRole map[string][]string
	if err := json.Unmarshal(data, &byRole); err != nil {
		return fmt.Errorf("parsing role assignments: %w", err)
	}

	builders := make(map[string]bool)
	for _, uid := range byRole[baseBuilderRoleID] {
		builders[uid] = true
	}
	for _, uid := range byRole[soloBuilderRoleID] {
		builders[uid] = true
	}

	var inactive []string
	for uid := range builders {
		if state.Counts[uid] < threshold {
			inactive = append(inactive, fmt.Sprintf("%s (%s) - %d messages", displayName(users, uid), uid, state.Counts[uid]))
		}
	}
	sort.Strings(inactive)

	slog.Info("builders below chat activity threshold", "count", len(inactive), "threshold", threshold, "report", reportPath)
	content := fmt.Sprintf("Builders below %d messages across tracked channels (%d total):\n\n%s\n", threshold, len(inactive), strings.Join(inactive, "\n"))
	return os.WriteFile(reportPath, []byte(content), 0o644)
}
