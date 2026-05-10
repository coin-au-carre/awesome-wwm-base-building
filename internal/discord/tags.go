package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

type tagsConfig struct {
	Guild []string `json:"guild"`
	Solo  []string `json:"solo"`
}

func loadTagsConfig(root string) (tagsConfig, error) {
	path := filepath.Join(root, "config", "tags.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return tagsConfig{}, fmt.Errorf("read tags config: %w", err)
	}
	var cfg tagsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return tagsConfig{}, fmt.Errorf("parse tags config: %w", err)
	}
	return cfg, nil
}

func handleSyncTagsCommand(s *discordgo.Session, i *discordgo.InteractionCreate, root, guildForumID, soloForumID string) {
	cfg, err := loadTagsConfig(root)
	if err != nil {
		slog.Error("loading tags config", "err", err)
		respondEphemeral(s, i, "*(couldn't load config/tags.json — check the file exists and is valid JSON.)*")
		return
	}

	type forumJob struct {
		label    string
		id       string
		tags     []string
		jsonPath string
	}
	jobs := []forumJob{
		{"guild", guildForumID, cfg.Guild, filepath.Join(root, "data", "guilds.json")},
		{"solo", soloForumID, cfg.Solo, filepath.Join(root, "data", "solos.json")},
	}

	var lines []string
	for _, job := range jobs {
		if job.id == "" {
			continue
		}

		tagSummary, nameToID, err := SyncForumTags(s, job.id, job.tags)
		if err != nil {
			slog.Error("syncing forum tags", "forum", job.label, "err", err)
			lines = append(lines, fmt.Sprintf("**%s** tags: error — %s", job.label, err))
			continue
		}
		slog.Info("forum tags synced", "forum", job.label, "result", tagSummary)

		reapplied, skipped, err := reapplyThreadTags(s, job.jsonPath, nameToID)
		if err != nil {
			slog.Error("reapplying thread tags", "forum", job.label, "err", err)
			lines = append(lines, fmt.Sprintf("**%s** tags: %s (thread reapply failed: %s)", job.label, tagSummary, err))
			continue
		}
		slog.Info("thread tags reapplied", "forum", job.label, "reapplied", reapplied, "skipped", skipped)
		lines = append(lines, fmt.Sprintf("**%s**: %s — threads updated: %d (skipped: %d)", job.label, tagSummary, reapplied, skipped))
	}

	if len(lines) == 0 {
		respondEphemeral(s, i, "*(no forum channel IDs configured.)*")
		return
	}

	msg := "Tag sync complete:\n"
	for _, l := range lines {
		msg += "• " + l + "\n"
	}
	respondEphemeral(s, i, msg)
}

// SyncForumTags updates the available tags on a forum channel to match the
// canonical list. Existing tags whose names match are preserved (keeping their
// Discord ID and emoji); tags not in the list are removed; new names are added.
// Returns a human-readable summary and the resulting name→ID map.
func SyncForumTags(s *discordgo.Session, channelID string, canonical []string) (summary string, nameToID map[string]string, err error) {
	ch, err := s.Channel(channelID)
	if err != nil {
		return "", nil, fmt.Errorf("fetch channel: %w", err)
	}

	currentByName := make(map[string]discordgo.ForumTag, len(ch.AvailableTags))
	for _, t := range ch.AvailableTags {
		currentByName[t.Name] = t
	}

	canonicalSet := make(map[string]bool, len(canonical))
	for _, name := range canonical {
		canonicalSet[name] = true
	}

	var added, removed []string
	newTags := make([]discordgo.ForumTag, 0, len(canonical))
	for _, name := range canonical {
		if existing, ok := currentByName[name]; ok {
			newTags = append(newTags, existing)
		} else {
			newTags = append(newTags, discordgo.ForumTag{Name: name})
			added = append(added, name)
		}
	}
	for _, t := range ch.AvailableTags {
		if !canonicalSet[t.Name] {
			removed = append(removed, t.Name)
		}
	}

	if len(added) > 0 || len(removed) > 0 {
		updated, err := s.ChannelEdit(channelID, &discordgo.ChannelEdit{
			AvailableTags: &newTags,
		})
		if err != nil {
			return "", nil, fmt.Errorf("update channel tags: %w", err)
		}
		// Use the IDs Discord assigned to newly created tags.
		newTags = updated.AvailableTags
	}

	nameToID = make(map[string]string, len(newTags))
	for _, t := range newTags {
		nameToID[t.Name] = t.ID
	}

	if len(added) == 0 && len(removed) == 0 {
		return "already up to date", nameToID, nil
	}
	parts := []string{}
	if len(added) > 0 {
		parts = append(parts, fmt.Sprintf("added: %v", added))
	}
	if len(removed) > 0 {
		parts = append(parts, fmt.Sprintf("removed: %v", removed))
	}
	return strings.Join(parts, " | "), nameToID, nil
}

// patchThreadTags unarchives the thread if needed, sets applied tags, then re-archives it.
func patchThreadTags(s *discordgo.Session, threadID string, tagIDs []string) error {
	ch, err := s.Channel(threadID)
	if err != nil {
		return fmt.Errorf("fetch thread: %w", err)
	}

	if ch.ThreadMetadata != nil && ch.ThreadMetadata.Archived {
		f := false
		if _, err := s.ChannelEdit(threadID, &discordgo.ChannelEdit{Archived: &f}); err != nil {
			return fmt.Errorf("unarchive: %w", err)
		}
	}

	if _, err := s.ChannelEdit(threadID, &discordgo.ChannelEdit{AppliedTags: &tagIDs}); err != nil {
		return fmt.Errorf("set tags: %w", err)
	}

	if ch.ThreadMetadata != nil && ch.ThreadMetadata.Archived {
		t := true
		if _, err := s.ChannelEdit(threadID, &discordgo.ChannelEdit{Archived: &t}); err != nil {
			slog.Warn("re-archiving thread after tag update", "thread", threadID, "err", err)
		}
	}

	return nil
}

// reapplyThreadTags reads guild/solo JSON, and for each entry with a discordThread URL
// patches the thread's applied_tags to match the tags stored in the JSON.
// Returns counts of threads updated and skipped (no tags or no thread URL).
func reapplyThreadTags(s *discordgo.Session, jsonPath string, nameToID map[string]string) (reapplied, skipped int, err error) {
	items, err := guild.LoadFile(jsonPath)
	if err != nil {
		return 0, 0, fmt.Errorf("load json: %w", err)
	}

	for _, item := range items {
		if item.DiscordThread == "" {
			skipped++
			continue
		}

		// Extract thread ID from URL: https://discord.com/channels/{guildID}/{threadID}
		parts := strings.Split(item.DiscordThread, "/")
		threadID := parts[len(parts)-1]

		tagIDs := make([]string, 0, len(item.Tags))
		for _, name := range item.Tags {
			if id, ok := nameToID[name]; ok {
				tagIDs = append(tagIDs, id)
			}
		}

		if err := patchThreadTags(s, threadID, tagIDs); err != nil {
			slog.Warn("reapplying tags to thread", "thread", threadID, "name", item.Name, "err", err)
			skipped++
			continue
		}
		reapplied++
	}

	return reapplied, skipped, nil
}
