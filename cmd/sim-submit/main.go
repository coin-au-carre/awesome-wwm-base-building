// cmd/sim-submit/main.go — simulate the DM Ruby sends after /submit-guild.
// Sends the formatted post to the dev channel instead of a real DM.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"ruby/internal/cmdutil"
)

var (
	reLocBracket = regexp.MustCompile(`^(.*?)\s*[\[(](?:ID\s+)?(\d+)[\])]?\s*$`)
	reLocSpaceID = regexp.MustCompile(`^(.*?)\s+(\d+)\s*$`)
	reSlugify    = regexp.MustCompile(`[^\p{L}\p{N}]+`)
)

func parseLocation(loc string) (name, id string) {
	loc = strings.TrimSpace(loc)
	if m := reLocBracket.FindStringSubmatch(loc); len(m) == 3 {
		return strings.TrimSpace(m[1]), m[2]
	}
	if m := reLocSpaceID.FindStringSubmatch(loc); len(m) == 3 {
		return strings.TrimSpace(m[1]), m[2]
	}
	return loc, ""
}

func slugify(name string) string {
	return strings.Trim(reSlugify.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

const websiteBase = "https://www.wherebuildersmeet.com"

func main() {
	nameFlag := flag.String("name", "Iron Vanguard [12345678]", "guild name (with optional [GuildID])")
	builders := flag.String("builders", "BuilderOne, BuilderTwo", "comma-separated builder names")
	lore := flag.String("lore", "A stronghold carved into volcanic rock, built to guard the eastern pass.", "lore text")
	whatToVisit := flag.String("what-to-visit", "- The lava forge\n- The obsidian throne room\n- The outer ramparts", "what to visit text")
	channelID := flag.String("channel", "", "Discord channel ID (default: DEV_CHANNEL_ID)")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(cmdutil.RootDir(), ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	target := *channelID
	if target == "" {
		target = os.Getenv("DEV_CHANNEL_ID")
	}
	if target == "" {
		slog.Error("provide -channel or set DEV_CHANNEL_ID")
		os.Exit(1)
	}

	guildForumChannelID := os.Getenv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")
	channelMention := "<#" + guildForumChannelID + ">"
	if guildForumChannelID == "" {
		channelMention = "**#guild-base-showcase**"
	}

	parsedName, parsedID := parseLocation(*nameFlag)
	threadTitle := parsedName
	if parsedID != "" {
		threadTitle = fmt.Sprintf("%s [%s]", parsedName, parsedID)
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("## 🏯 %s\n\n", threadTitle))
	if *builders != "" {
		content.WriteString(fmt.Sprintf("👷 Builders: %s\n\n", *builders))
	}
	if *lore != "" {
		content.WriteString(fmt.Sprintf("### 📝 Lore\n%s\n\n", *lore))
	}
	if *whatToVisit != "" {
		content.WriteString(fmt.Sprintf("### 🧙 What to visit\n%s", *whatToVisit))
	}

	dm := fmt.Sprintf(
		"## 🏯 %s\n\n"+
			"Here's your formatted post, ready to copy!\n\n"+
			"**1.** Go to %s\n"+
			"**2.** Create a new post titled: `%s`\n"+
			"**3.** Paste the text below as your message\n"+
			"**4.** Add your screenshots 📸 (10 max for the first post) you can always add more later\n\n"+
			"**5.** Submit your post! \n"+
			"```\n%s\n```",
		parsedName, channelMention, parsedName, strings.TrimSpace(content.String()),
	)

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}

	if _, err := s.ChannelMessageSend(target, dm); err != nil {
		slog.Error("sending message", "err", err)
		os.Exit(1)
	}

	slog.Info("sent sim-submit message", "channel", target, "guild", parsedName)
}
