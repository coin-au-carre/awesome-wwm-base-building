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
	link := flag.String("url", "", "Discord message URL (https://discord.com/channels/{guild}/{channel}/{message})")
	flag.Parse()

	if *link == "" {
		*link = flag.Arg(0)
	}
	if *link == "" {
		slog.Error("usage: task numreact -- -url <discord-message-url> -n <N>")
		os.Exit(1)
	}
	if *n < 1 || *n > 10 {
		slog.Error("-n must be between 1 and 10", "got", *n)
		os.Exit(1)
	}

	// Parse channel and message IDs from:
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
	channelID := parts[1]
	messageID := parts[2]

	cmdutil.LoadEnv(cmdutil.RootDir())
	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating session", "err", err)
		os.Exit(1)
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
