package main

import (
	"flag"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
)

// :one: through :ten: Unicode keycap/number emojis.
var numberEmojis = []string{
	"1️⃣", // :one:
	"2️⃣", // :two:
	"3️⃣", // :three:
	"4️⃣", // :four:
	"5️⃣", // :five:
	"6️⃣", // :six:
	"7️⃣", // :seven:
	"8️⃣", // :eight:
	"9️⃣", // :nine:
	"🔟",  // :ten:
}

func main() {
	n := flag.Int("n", 0, "add reactions :one: through :N: (1–10)")
	allServerEmojis := flag.Bool("all-server-emojis", false, "react with every custom emoji in the server (animated and non-animated)")
	link := flag.String("url", "", "Discord message URL (https://discord.com/channels/{guild}/{channel}/{message})")
	flag.Parse()

	if *link == "" {
		*link = flag.Arg(0)
	}
	if *link == "" {
		slog.Error("usage: task numreact -- -url <discord-message-url> [-n <N>] [-all-server-emojis]")
		os.Exit(1)
	}
	if !*allServerEmojis && (*n < 1 || *n > 10) {
		slog.Error("-n must be between 1 and 10, or use -all-server-emojis", "got", *n)
		os.Exit(1)
	}

	// Parse guild, channel, and message IDs from:
	// https://discord.com/channels/{guild}/{channel}/{message}
	u, err := url.Parse(*link)
	if err != nil {
		slog.Error("parsing URL", "err", err)
		os.Exit(1)
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/channels/"), "/")
	if len(parts) != 3 {
		slog.Error("expected URL format: https://discord.com/channels/{guild}/{channel}/{message}")
		os.Exit(1)
	}
	guildID := parts[0]
	channelID := parts[1]
	messageID := parts[2]

	cmdutil.LoadEnv(cmdutil.RootDir())
	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
	}

	if *allServerEmojis {
		emojis, err := s.GuildEmojis(guildID)
		if err != nil {
			slog.Error("fetching guild emojis", "err", err)
			os.Exit(1)
		}
		slog.Info("fetched server emojis", "count", len(emojis))
		added := 0
		for _, e := range emojis {
			apiName := e.APIName()
			if err := s.MessageReactionAdd(channelID, messageID, apiName); err != nil {
				slog.Warn("reaction failed", "emoji", e.Name, "err", err)
			} else {
				slog.Info("reacted", "emoji", e.Name, "animated", e.Animated)
				added++
			}
			time.Sleep(300 * time.Millisecond)
		}
		slog.Info("done", "reactions_added", added, "total_emojis", len(emojis))
		return
	}

	for i := 0; i < *n; i++ {
		emoji := numberEmojis[i]
		if err := s.MessageReactionAdd(channelID, messageID, emoji); err != nil {
			slog.Error("adding reaction", "n", i+1, "emoji", emoji, "err", err)
			os.Exit(1)
		}
		slog.Info("reacted", "n", i+1, "emoji", emoji)
		time.Sleep(300 * time.Millisecond)
	}

	slog.Info("done", "reactions_added", *n)
}
