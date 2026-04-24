package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"ruby/internal/guild"

	"github.com/bwmarrin/discordgo"
)

func resolveUsername(s *discordgo.Session, userID string) string {
	if u, err := s.User(userID); err == nil {
		return u.Username
	}
	return userID
}

// RoleCache tracks which users have already been assigned each role so that
// syncs skip redundant Discord API calls.
type RoleCache struct {
	path    string
	entries map[string]map[string]bool // roleID → set of userIDs
	dirty   bool
}

// LoadRoleCache reads the cache from path. Missing file is treated as empty.
func LoadRoleCache(path string) (*RoleCache, error) {
	c := &RoleCache{path: path, entries: make(map[string]map[string]bool)}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading role cache: %w", err)
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing role cache: %w", err)
	}
	for roleID, users := range raw {
		c.entries[roleID] = make(map[string]bool, len(users))
		for _, uid := range users {
			c.entries[roleID][uid] = true
		}
	}
	return c, nil
}

// Has reports whether userID already has roleID recorded in the cache.
func (c *RoleCache) Has(roleID, userID string) bool {
	return c.entries[roleID][userID]
}

// mark records a successful assignment in the cache.
func (c *RoleCache) mark(roleID, userID string) {
	if c.entries[roleID] == nil {
		c.entries[roleID] = make(map[string]bool)
	}
	c.entries[roleID][userID] = true
	c.dirty = true
}

// Save writes the cache to disk if anything changed.
func (c *RoleCache) Save() error {
	if !c.dirty {
		return nil
	}
	raw := make(map[string][]string, len(c.entries))
	for roleID, users := range c.entries {
		list := make([]string, 0, len(users))
		for uid := range users {
			list = append(list, uid)
		}
		raw[roleID] = list
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o644)
}

// AssignAwesomeBuilderRole grants the awesome Builder role to a thread author.
// Skips the API call if cache already records the assignment.
func AssignAwesomeBuilderRole(s *discordgo.Session, guildID, userID, roleID, baseName string, cache *RoleCache) {
	if roleID == "" || userID == "" || guildID == "" {
		return
	}
	if cache != nil && cache.Has(roleID, userID) {
		return
	}
	if err := s.GuildMemberRoleAdd(guildID, userID, roleID); err != nil {
		slog.Warn("assigning awesome builder role failed", "guild", baseName, "userID", userID, "user", resolveUsername(s, userID), "err", err)
		return
	}
	slog.Info("awesome builder role assigned", "guild", baseName, "user", resolveUsername(s, userID), "id", userID)
	if cache != nil {
		cache.mark(roleID, userID)
	}
}

// AssignRoleToVoters assigns roleID to users who voted on at least minVotes distinct guilds.
func AssignRoleToVoters(s *discordgo.Session, discordGuildID, roleID string, voterGuildCounts map[string]int, minVotes int, cache *RoleCache) {
	assigned := 0
	for uid, count := range voterGuildCounts {
		if count < minVotes {
			continue
		}
		if cache != nil && cache.Has(roleID, uid) {
			continue
		}
		if err := s.GuildMemberRoleAdd(discordGuildID, uid, roleID); err != nil {
			slog.Warn("assigning critic role failed", "user", uid, "err", err)
			continue
		}
		slog.Info("critic role assigned", "user", resolveUsername(s, uid), "id", uid, "guilds_voted", count)
		if cache != nil {
			cache.mark(roleID, uid)
		}
		assigned++
	}
	slog.Info("critic role assignment done", "assigned", assigned, "min_votes", minVotes)
}

// AssignRoleByScore assigns roleID to any guild author whose Score >= minScore.
func AssignRoleByScore(s *discordgo.Session, discordGuildID, roleID string, guilds []guild.Guild, minScore int, skipUsers map[string]bool, cache *RoleCache) {
	assigned := make(map[string]bool)
	for _, g := range guilds {
		userID := g.PosterDiscordID
		if userID == "" || assigned[userID] || skipUsers[userID] {
			continue
		}
		if g.Score >= minScore {
			AssignAwesomeBuilderRole(s, discordGuildID, userID, roleID, g.Name, cache)
			assigned[userID] = true
		}
	}
}

// AssignRoleToForumAuthors fetches all threads in forumChannelID and assigns
// roleID to each thread's original poster, skipping any user ID in skipUsers.
// Pass nil skipUsers to assign everyone (e.g. with --force-role).
func AssignRoleToForumAuthors(s *discordgo.Session, forumChannelID, roleID string, skipUsers map[string]bool, cache *RoleCache) error {
	if forumChannelID == "" || roleID == "" {
		return nil
	}
	ch, err := s.Channel(forumChannelID)
	if err != nil {
		return fmt.Errorf("fetching channel %s: %w", forumChannelID, err)
	}
	guildID := ch.GuildID

	threads, err := collectThreads(s, forumChannelID, guildID)
	if err != nil {
		return err
	}

	assigned := make(map[string]bool)
	for _, thread := range threads {
		msgs, err := s.ChannelMessages(thread.ID, 1, "", "0", "")
		if err != nil || len(msgs) == 0 {
			slog.Warn("fetching thread first message", "thread", thread.ID, "err", err)
			continue
		}
		authorID := msgs[0].Author.ID
		if assigned[authorID] || skipUsers[authorID] {
			continue
		}
		AssignAwesomeBuilderRole(s, guildID, authorID, roleID, thread.Name, cache)
		assigned[authorID] = true
	}
	return nil
}
