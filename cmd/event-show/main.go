// cmd/event-show/main.go — post upcoming events (next 4 hours) to a Discord channel.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/guild"
)

func main() {
	dev := flag.Bool("dev", false, "post to DEV_CHANNEL_ID instead of GENERAL_CHANNEL_ID")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(cmdutil.RootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	guildID := cmdutil.RequireEnv("DISCORD_GUILD_ID")

	channelID := os.Getenv("GENERAL_CHANNEL_ID")
	if *dev {
		channelID = cmdutil.RequireEnv("DEV_CHANNEL_ID")
	} else if channelID == "" {
		slog.Error("GENERAL_CHANNEL_ID not set")
		os.Exit(1)
	}

	root := cmdutil.RootDir()

	guilds, err := guild.Load(root)
	if err != nil {
		slog.Warn("could not load guilds", "err", err)
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating discord session", "err", err)
		os.Exit(1)
	}

	events, err := discord.FetchEvents(s, guildID)
	if err != nil {
		slog.Error("fetching events", "err", err)
		os.Exit(1)
	}

	now := time.Now()
	horizon := now.Add(4 * time.Hour)

	var upcoming []discord.Event
	for _, e := range events {
		if e.Status != discord.EventStatusScheduled && e.Status != discord.EventStatusActive {
			continue
		}
		if e.ScheduledStart.After(now) && e.ScheduledStart.Before(horizon) {
			upcoming = append(upcoming, e)
		}
	}

	if len(upcoming) == 0 {
		slog.Info("no events in the next 4 hours")
		return
	}

	msg := formatUpcomingEvents(upcoming, guilds, now)
	if _, err := s.ChannelMessageSend(channelID, msg); err != nil {
		slog.Error("sending message", "err", err)
		os.Exit(1)
	}
	slog.Info("posted upcoming events", "count", len(upcoming), "channel", channelID)
}

// findGuildThread returns the DiscordThread URL for a guild matching the event's
// GuildID (exact) or GuildName (case-insensitive substring).
func findGuildThread(guilds []guild.Guild, guildID, guildName string) string {
	for _, g := range guilds {
		if guildID != "" && g.ID == guildID {
			return g.DiscordThread
		}
	}
	if guildName == "" {
		return ""
	}
	lower := strings.ToLower(guildName)
	for _, g := range guilds {
		if strings.Contains(strings.ToLower(g.Name), lower) {
			return g.DiscordThread
		}
	}
	return ""
}

func formatUpcomingEvents(events []discord.Event, guilds []guild.Guild, now time.Time) string {
	var sb strings.Builder
	sb.WriteString("📅 **Upcoming events in the next 4 hours:**\n\n")
	for _, e := range events {
		in := time.Until(e.ScheduledStart).Round(time.Minute)
		var inStr string
		if in <= 0 {
			inStr = "starting now"
		} else {
			h := int(in.Hours())
			m := int(in.Minutes()) % 60
			if h > 0 {
				inStr = fmt.Sprintf("in %dh%02dm", h, m)
			} else {
				inStr = fmt.Sprintf("in %dm", m)
			}
		}
		sb.WriteString(fmt.Sprintf("**[%s](<%s>)** — %s", e.Name, e.DiscordURL, inStr))
		if e.GuildName != "" {
			thread := findGuildThread(guilds, e.GuildID, e.GuildName)
			if thread != "" {
				sb.WriteString(fmt.Sprintf(" • [%s](<%s>)", e.GuildName, thread))
			} else {
				sb.WriteString(fmt.Sprintf(" • %s", e.GuildName))
			}
		}
		sb.WriteString("\n")
		if e.Description != "" {
			sb.WriteString(fmt.Sprintf("> %s\n", e.Description))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
