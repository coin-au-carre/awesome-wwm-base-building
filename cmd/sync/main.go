// cmd/sync/main.go
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ruby/internal/blueprint"
	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "crawl Discord but skip writing JSON files")
	noNotify := flag.Bool("no-notify", false, "skip posting summary to bot channel")
	forceRole := flag.Bool("force-role", false, "reassign roles to all authors, ignoring the role cache")
	root := flag.String("root", cmdutil.RootDir(), "root directory containing guilds.json and solos.json")
	guildFilter := flag.String("guild", "", "only sync threads whose name contains this string (case-insensitive)")
	only := flag.String("only", "", "sync only specific channels: guilds, solo, blueprints (comma-separated; default: all)")
	flag.Parse()

	syncGuilds := true
	syncSolo := true
	syncBlueprints := true
	if *only != "" {
		syncGuilds, syncSolo, syncBlueprints = false, false, false
		for _, ch := range strings.Split(*only, ",") {
			switch strings.TrimSpace(strings.ToLower(ch)) {
			case "guilds", "guild":
				syncGuilds = true
			case "solo", "solos":
				syncSolo = true
			case "blueprints", "blueprint":
				syncBlueprints = true
			}
		}
	}

	cmdutil.LoadEnv(*root)

	var guildForumID string
	if syncGuilds {
		guildForumID = cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	}
	soloForumID := os.Getenv("SOLO_BUILD_SHOWCASE_CHANNEL_FORUM_ID")
	blueprintForumID := os.Getenv("BLUEPRINT_CHANNEL_FORUM_ID")
	botChannelID := os.Getenv("BOT_CHANNEL_ID")
	devChannelID := os.Getenv("DEV_CHANNEL_ID")
	generalChannelID := os.Getenv("GENERAL_CHANNEL_ID")
	baseBuilderRoleID := os.Getenv("BASE_BUILDER_ROLE_ID")
	baseCriticRoleID := os.Getenv("BASE_CRITIC_ROLE_ID")

	guildsPath := filepath.Join(*root, "data", "guilds.json")
	solosPath := filepath.Join(*root, "data", "solos.json")
	blueprintsPath := filepath.Join(*root, "data", "blueprints.json")
	roleCachePath := filepath.Join(*root, "data", "role_assignments.json")
	blacklistPath := filepath.Join(*root, "data", "voter_blacklist.json")
	whitelistPath := filepath.Join(*root, "data", "voter_whitelist.json")

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	bot, err := discord.NewBot(token, botChannelID)
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()

	if err := bot.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	discordTriggered := os.Getenv("DISCORD_TRIGGER") == "true"
	progressMsgID := ""
	if !*noNotify && !discordTriggered && botChannelID != "" {
		progressMsgID = bot.SendReturnID(botChannelID, "🔄 *(slipping through guild halls...)*")
	}

	// ── Load existing data ────────────────────────────────────────────────────

	var guilds []guild.Guild
	if syncGuilds {
		guilds, err = guild.LoadFile(guildsPath)
		if err != nil {
			slog.Warn("could not load guilds, starting fresh", "err", err)
			guilds = []guild.Guild{}
		}
	}

	var solos []guild.Guild
	if syncSolo && soloForumID != "" {
		solos, err = guild.LoadFile(solosPath)
		if err != nil {
			slog.Warn("could not load solos, starting fresh", "err", err)
			solos = []guild.Guild{}
		}
	}

	var blueprints []blueprint.Blueprint
	if syncBlueprints && blueprintForumID != "" {
		blueprints, err = blueprint.LoadFile(blueprintsPath)
		if err != nil {
			slog.Warn("could not load blueprints, starting fresh", "err", err)
			blueprints = []blueprint.Blueprint{}
		}
	}

	blacklist, err := guild.LoadVoterBlacklist(blacklistPath)
	if err != nil {
		slog.Error("loading voter blacklist", "err", err)
		os.Exit(1)
	}
	slog.Info("voter blacklist loaded", "count", len(blacklist))

	whitelist, err := guild.LoadVoterBlacklist(whitelistPath)
	if err != nil {
		slog.Warn("could not load voter whitelist", "err", err)
	}
	slog.Info("voter whitelist loaded", "count", len(whitelist))

	// ── Fetch channels in parallel ────────────────────────────────────────────

	type fetchOutcome struct {
		result discord.SyncFetchResult
		err    error
	}
	type blueprintFetchOutcome struct {
		result discord.BlueprintSyncFetchResult
		err    error
	}

	guildCh := make(chan fetchOutcome, 1)
	soloCh := make(chan fetchOutcome, 1)
	blueprintCh := make(chan blueprintFetchOutcome, 1)

	makeProgressFn := func(label string) func(done, total int) {
		lastPct := -1
		return func(done, total int) {
			if progressMsgID == "" {
				return
			}
			pct := done * 100 / total
			if pct/10 == lastPct/10 {
				return
			}
			lastPct = pct
			bar := discord.ProgressBar(pct)
			bot.EditMessage(botChannelID, progressMsgID, fmt.Sprintf(
				"🔄 *(wandering through %s...)* %s %d%%", label, bar, pct,
			))
		}
	}

	var fetchWg sync.WaitGroup
	if syncGuilds {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			r, err := discord.SyncFetch(bot, guilds, discord.SyncConfig{
				ForumChannelID: guildForumID,
				GuildFilter:    *guildFilter,
				OnProgress:     makeProgressFn("guild halls"),
			})
			guildCh <- fetchOutcome{r, err}
		}()
	}

	if syncSolo && soloForumID != "" {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			r, err := discord.SyncFetch(bot, solos, discord.SyncConfig{
				ForumChannelID: soloForumID,
				IsSolo:         true,
				GuildFilter:    *guildFilter,
				OnProgress:     makeProgressFn("solo courts"),
			})
			soloCh <- fetchOutcome{r, err}
		}()
	}

	if syncBlueprints && blueprintForumID != "" {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			r, err := discord.BlueprintSyncFetch(bot, blueprints, discord.SyncConfig{
				ForumChannelID: blueprintForumID,
				GuildFilter:    *guildFilter,
				OnProgress:     makeProgressFn("blueprint hall"),
			})
			blueprintCh <- blueprintFetchOutcome{r, err}
		}()
	}

	fetchWg.Wait()
	close(guildCh)
	close(soloCh)
	close(blueprintCh)
	if progressMsgID != "" {
		bot.EditMessage(botChannelID, progressMsgID, "🔄 *(counting stars and sealing the scrolls...)*")
	}

	var guildFetch fetchOutcome
	if syncGuilds {
		guildFetch = <-guildCh
		if guildFetch.err != nil {
			bot.NotifyIf(!*noNotify, "💥 **Guilds failed to synchronize!** — "+guildFetch.err.Error())
			slog.Error("guild fetch failed", "err", guildFetch.err)
			os.Exit(1)
		}
	}

	var soloFetch fetchOutcome
	if syncSolo && soloForumID != "" {
		soloFetch = <-soloCh
		if soloFetch.err != nil {
			bot.NotifyIf(!*noNotify, "💥 **Solo construction failed to synchronize!** — "+soloFetch.err.Error())
			slog.Error("solo fetch failed", "err", soloFetch.err)
			os.Exit(1)
		}
	}

	var blueprintFetch blueprintFetchOutcome
	if syncBlueprints && blueprintForumID != "" {
		blueprintFetch = <-blueprintCh
		if blueprintFetch.err != nil {
			if !*noNotify && devChannelID != "" {
				bot.Send(devChannelID, "💥 **Blueprints failed to synchronize!** — "+blueprintFetch.err.Error())
			}
			slog.Error("blueprint fetch failed", "err", blueprintFetch.err)
			os.Exit(1)
		}
	}

	// ── Finalize (score) ─────────────────────────────────────────────────────

	var updatedGuilds []guild.Guild
	var guildReactions guild.ReactionMap
	var guildStats discord.SyncStats
	if syncGuilds {
		guildWeights := discord.ComputeVoterWeights(guildFetch.result.VoterCounts)
		slog.Info("guild voter weights", "voters", len(guildWeights))
		updatedGuilds, guildReactions, guildStats = discord.SyncFinalize(guildFetch.result, guildWeights, blacklist, whitelist)
	}

	var updatedSolos []guild.Guild
	var soloReactions guild.ReactionMap
	var soloStats discord.SyncStats
	if syncSolo && soloForumID != "" {
		soloWeights := discord.ComputeVoterWeights(soloFetch.result.VoterCounts)
		slog.Info("solo voter weights", "voters", len(soloWeights))
		updatedSolos, soloReactions, soloStats = discord.SyncFinalize(soloFetch.result, soloWeights, blacklist, whitelist)
	}

	var updatedBlueprints []blueprint.Blueprint
	var blueprintStats discord.SyncStats
	if syncBlueprints && blueprintForumID != "" {
		bpWeights := discord.ComputeVoterWeights(blueprintFetch.result.VoterCounts)
		slog.Info("blueprint voter weights", "voters", len(bpWeights))
		updatedBlueprints, blueprintStats = discord.BlueprintSyncFinalize(blueprintFetch.result, bpWeights, blacklist)
	}

	// ── Save ──────────────────────────────────────────────────────────────────

	if *dryRun {
		slog.Info("dry-run: skipping save")
	} else {
		if syncGuilds {
			if err := guild.SaveFile(guildsPath, updatedGuilds); err != nil {
				slog.Error("saving guilds", "err", err)
				os.Exit(1)
			}
			if guildStats.New > 0 {
				cmdutil.UpdateNavVersion(*root, "guilds")
			}
		}
		if syncSolo && soloForumID != "" {
			if err := guild.SaveFile(solosPath, updatedSolos); err != nil {
				slog.Error("saving solos", "err", err)
				os.Exit(1)
			}
			if soloStats.New > 0 {
				cmdutil.UpdateNavVersion(*root, "solo")
			}
		}
		if syncBlueprints && blueprintForumID != "" {
			if err := blueprint.SaveFile(blueprintsPath, updatedBlueprints); err != nil {
				slog.Error("saving blueprints", "err", err)
				os.Exit(1)
			}
			if blueprintStats.New > 0 {
				cmdutil.UpdateNavVersion(*root, "blueprints")
			}
		}

		allReactions := make(guild.ReactionMap)
		for k, v := range guildReactions {
			allReactions[k] = v
		}
		for k, v := range soloReactions {
			allReactions[k] = v
		}
		if len(allReactions) > 0 {
			if err := guild.SaveReactions(*root, allReactions); err != nil {
				slog.Warn("saving reactions", "err", err)
			}
		}

		allUsers, _ := guild.LoadUsers(*root)
		if allUsers == nil {
			allUsers = make(guild.UserMap)
		}
		for k, v := range guildFetch.result.Users {
			allUsers[k] = v
		}
		for k, v := range soloFetch.result.Users {
			allUsers[k] = v
		}
		for k, v := range blueprintFetch.result.Users {
			allUsers[k] = v
		}

		// Resolve scoutedByDiscordId values not already covered by voter/author resolution.
		var scoutIDs []string
		seen := make(map[string]bool)
		for _, g := range updatedGuilds {
			if id := g.ScoutedByDiscordID; id != "" && !seen[id] && allUsers[id].Username == "" {
				scoutIDs = append(scoutIDs, id)
				seen[id] = true
			}
		}
		for _, g := range updatedSolos {
			if id := g.ScoutedByDiscordID; id != "" && !seen[id] && allUsers[id].Username == "" {
				scoutIDs = append(scoutIDs, id)
				seen[id] = true
			}
		}
		if len(scoutIDs) > 0 {
			discordGuildID := ""
			if guildForumID != "" {
				if ch, err := bot.Session.Channel(guildForumID); err == nil {
					discordGuildID = ch.GuildID
				}
			} else if soloForumID != "" {
				if ch, err := bot.Session.Channel(soloForumID); err == nil {
					discordGuildID = ch.GuildID
				}
			}
			if discordGuildID != "" {
				slog.Info("resolving scout user IDs", "count", len(scoutIDs))
				for k, v := range discord.ResolveUserIDs(bot.Session, discordGuildID, scoutIDs) {
					allUsers[k] = v
				}
			}
		}

		if len(allUsers) > 0 {
			if err := guild.SaveUsers(*root, allUsers); err != nil {
				slog.Warn("saving users", "err", err)
			}
		}

		lastSyncPath := filepath.Join(*root, "data", "last_sync.json")
		syncedAt := time.Now().UTC().Format(time.RFC3339)
		if err := os.WriteFile(lastSyncPath, []byte(fmt.Sprintf(`{"syncedAt":%q}`, syncedAt)+"\n"), 0644); err != nil {
			slog.Warn("writing last_sync.json", "err", err)
		}
	}

	// ── Guide replies for malformed new guild threads ─────────────────────────

	if syncGuilds && !*dryRun {
		const guideURL = "https://www.wherebuildersmeet.com/contribute/builder/"
		for _, threadID := range guildStats.MalformedNewThreadIDs {
			msg := "👋 Hey! It looks like your post is missing a guild name or guild ID.\n" +
				"Check the submission guide for the right format: " + guideURL + "\n" +
				"You can also use **/submit-guild** to get a ready-to-paste template sent to your DMs."
			bot.Send(threadID, msg)
			slog.Info("sent guide reply to malformed guild thread", "thread", threadID)
		}
	}

	// ── Role assignment ───────────────────────────────────────────────────────

	if syncGuilds && !*dryRun && (baseBuilderRoleID != "" || baseCriticRoleID != "") {
		forumCh, err := bot.Session.Channel(guildForumID)
		if err != nil {
			slog.Warn("fetching forum channel for role assignment", "err", err)
		} else {
			roleCache, err := discord.LoadRoleCache(roleCachePath)
			if err != nil {
				slog.Warn("loading role cache, skipping cache", "err", err)
				roleCache = nil
			}
			if *forceRole {
				roleCache = nil
			}
			discordGuildID := forumCh.GuildID
			if baseBuilderRoleID != "" {
				discord.AssignRoleByScore(bot.Session, discordGuildID, baseBuilderRoleID, updatedGuilds, 0, nil, roleCache)
				if syncSolo && soloForumID != "" {
					discord.AssignRoleByScore(bot.Session, discordGuildID, baseBuilderRoleID, updatedSolos, 0, nil, roleCache)
				}
			}
			if baseCriticRoleID != "" {
				mergedCounts := discord.MergeVoterCounts(guildFetch.result.VoterCounts, soloFetch.result.VoterCounts)
				slog.Info("assigning critic role", "total_voters", len(mergedCounts))
				discord.AssignRoleToVoters(bot.Session, discordGuildID, baseCriticRoleID, mergedCounts, 6, roleCache)
			}
			if roleCache != nil {
				if err := roleCache.Save(); err != nil {
					slog.Warn("saving role cache", "err", err)
				}
			}
		}
	}

	// ── Summary ───────────────────────────────────────────────────────────────

	summary := discord.FormatCombinedSyncSummary(guildStats, soloStats, syncSolo && soloForumID != "")
	if syncBlueprints && blueprintForumID != "" && (blueprintStats.New > 0 || blueprintStats.Updated > 0) {
		summary += fmt.Sprintf("\n📐 **%d** blueprints — %d new, %d updated", blueprintStats.Total, blueprintStats.New, blueprintStats.Updated)
	}
	if progressMsgID != "" {
		bot.EditMessage(botChannelID, progressMsgID, summary)
	} else {
		bot.NotifyIf(!*noNotify, summary)
	}
	if syncGuilds {
		slog.Info("guild sync complete", "total", guildStats.Total, "new", guildStats.New, "updated", guildStats.Updated)
	}
	if syncSolo {
		slog.Info("solo sync complete", "total", soloStats.Total, "new", soloStats.New, "updated", soloStats.Updated)
	}
	if syncBlueprints {
		slog.Info("blueprint sync complete", "total", blueprintStats.Total, "new", blueprintStats.New, "updated", blueprintStats.Updated)
	}

	if !*dryRun {
		const maxNewAnnouncements = 5
		guildSpam := syncGuilds && guildStats.New > maxNewAnnouncements
		soloSpam := syncSolo && soloStats.New > maxNewAnnouncements

		if guildSpam || soloSpam {
			warn := fmt.Sprintf(
				"⚠️ **Announcement suppressed** — unusually high new-entry count (guilds: %d, solos: %d). Check that guilds.json loaded correctly before announcing.",
				guildStats.New, soloStats.New,
			)
			slog.Warn("announcement suppressed: too many new entries", "guilds_new", guildStats.New, "solos_new", soloStats.New)
			if !*noNotify {
				bot.NotifyIf(true, warn)
				if devChannelID != "" {
					bot.Send(devChannelID, warn)
				}
			}
		} else {
			if syncGuilds && !*noNotify && guildStats.New > 0 {
				notifyNewEntries(bot, updatedGuilds, guildStats, false)
			}
			if syncSolo && !*noNotify && soloStats.New > 0 {
				notifyNewEntries(bot, updatedSolos, soloStats, true)
			}
			if !*noNotify && generalChannelID != "" {
				if syncGuilds {
					announceToGeneral(bot, generalChannelID, updatedGuilds, guildStats, false)
				}
				if syncSolo {
					announceToGeneral(bot, generalChannelID, updatedSolos, soloStats, true)
				}
				if syncBlueprints && blueprintForumID != "" {
					announceBlueprintsToGeneral(bot, generalChannelID, updatedBlueprints, blueprintStats)
				}
			}
		}
	}

	if devChannelID != "" && !*noNotify {
		var allWarnings []string
		if syncGuilds {
			allWarnings = append(allWarnings, guildStats.DuplicateWarnings...)
		}
		if syncSolo {
			allWarnings = append(allWarnings, soloStats.DuplicateWarnings...)
		}
		for _, w := range allWarnings {
			bot.Send(devChannelID, w)
		}
	}
}


const (
	ahlyamID = "149790526076354561"
	windxpID = "721510597958828183"
	babeID   = "376950312721711118"
)

func announceToGeneral(bot *discord.Bot, channelID string, entries []guild.Guild, stats discord.SyncStats, isSolo bool) {
	byName := make(map[string]guild.Guild, len(entries))
	for _, g := range entries {
		byName[g.Name] = g
	}
	announce := func(name string, isSoloEntry bool, msgFn func(guild.Guild, bool) string) {
		g, ok := byName[name]
		if !ok {
			return
		}
		bot.Send(channelID, msgFn(g, isSoloEntry))
	}
	hasNewVideos := make(map[string]bool, len(stats.MoreVideoNames))
	for _, name := range stats.MoreVideoNames {
		hasNewVideos[name] = true
	}
	for _, name := range stats.MoreScreenshotNames {
		if hasNewVideos[name] {
			continue
		}
		g, ok := byName[name]
		if !ok || g.PosterDiscordID == ahlyamID {
			continue
		}
		if g.PosterDiscordID == babeID && g.Name != "PleasureSeeker" && g.GuildName != "PleasureSeeker" {
			continue
		}
		msg := discord.FormatMoreScreenshotsMessage(g, isSolo)
		if len(g.Screenshots) > 0 {
			msg += "\n" + g.Screenshots[len(g.Screenshots)-1]
		}
		bot.Send(channelID, msg)
	}
	for _, name := range stats.MoreVideoNames {
		g, ok := byName[name]
		if !ok || g.PosterDiscordID == ahlyamID {
			continue
		}
		if g.PosterDiscordID == babeID && g.Name != "PleasureSeeker" && g.GuildName != "PleasureSeeker" {
			continue
		}
		announce(name, isSolo, discord.FormatMoreVideosMessage)
	}
	for _, name := range stats.NewNames {
		g, ok := byName[name]
		if !ok || g.PosterDiscordID == ahlyamID || g.PosterDiscordID == windxpID || g.PosterDiscordID == babeID {
			continue
		}
		msg := discord.FormatNewGuildMessage(g, isSolo)
		if len(g.Screenshots) > 0 {
			msg += "\n" + g.Screenshots[0]
		}
		bot.Send(channelID, msg)
	}
}

func announceBlueprintsToGeneral(bot *discord.Bot, channelID string, blueprints []blueprint.Blueprint, stats discord.SyncStats) {
	byName := make(map[string]blueprint.Blueprint, len(blueprints))
	for _, bp := range blueprints {
		byName[bp.Name] = bp
	}
	hasNewVideos := make(map[string]bool, len(stats.MoreVideoNames))
	for _, name := range stats.MoreVideoNames {
		hasNewVideos[name] = true
	}
	for _, name := range stats.MoreScreenshotNames {
		if hasNewVideos[name] {
			continue
		}
		bp, ok := byName[name]
		if !ok {
			continue
		}
		msg := discord.FormatMoreBlueprintScreenshotsMessage(bp)
		if len(bp.Screenshots) > 0 {
			msg += "\n" + bp.Screenshots[len(bp.Screenshots)-1]
		}
		bot.Send(channelID, msg)
	}
	for _, name := range stats.MoreVideoNames {
		bp, ok := byName[name]
		if !ok {
			continue
		}
		bot.Send(channelID, discord.FormatMoreBlueprintVideosMessage(bp))
	}
	for _, name := range stats.NewNames {
		bp, ok := byName[name]
		if !ok {
			continue
		}
		msg := discord.FormatNewBlueprintMessage(bp)
		if len(bp.Screenshots) > 0 {
			msg += "\n" + bp.Screenshots[0]
		}
		bot.Send(channelID, msg)
	}
}

func notifyNewEntries(bot *discord.Bot, entries []guild.Guild, stats discord.SyncStats, isSolo bool) {
	byName := make(map[string]guild.Guild, len(entries))
	for _, g := range entries {
		byName[g.Name] = g
	}
	for _, name := range stats.NewNames {
		g, ok := byName[name]
		if !ok || g.PosterDiscordID == ahlyamID || g.PosterDiscordID == windxpID || g.PosterDiscordID == babeID {
			continue
		}
		bot.Notify(discord.FormatNewGuildMessage(g, isSolo))
	}
}
