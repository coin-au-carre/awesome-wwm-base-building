// One-shot: list members who hold ALL of the given role IDs.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ruby/internal/cmdutil"
)

func main() {
	roleIDs := flag.String("roles", "", "comma-separated role IDs; only members holding ALL of them are listed")
	guildID := flag.String("guild", "", "Discord guild ID (defaults to the guild owning GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID)")
	flag.Parse()

	roles := strings.Split(strings.TrimSpace(*roleIDs), ",")
	if len(roles) == 0 || roles[0] == "" {
		slog.Error("must pass -roles=id1,id2,...")
		os.Exit(1)
	}

	cmdutil.LoadEnv(cmdutil.RootDir())
	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("creating Discord session", "err", err)
		os.Exit(1)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	discordGuildID := *guildID
	if discordGuildID == "" {
		ch, err := session.Channel(cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID"))
		if err != nil {
			slog.Error("fetching forum channel", "err", err)
			os.Exit(1)
		}
		discordGuildID = ch.GuildID
	}

	var members []*discordgo.Member
	var after string
	for {
		page, err := session.GuildMembers(discordGuildID, after, 1000)
		if err != nil {
			slog.Error("fetching guild members", "err", err)
			os.Exit(1)
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

	matched := 0
	for _, m := range members {
		if hasAllRoles(m.Roles, roles) {
			name := m.User.Username
			if m.Nick != "" {
				name = m.Nick
			}
			fmt.Printf("%s (%s)\n", name, m.User.ID)
			matched++
		}
	}
	slog.Info("done", "scanned", len(members), "matched", matched, "roles", roles)
}

func hasAllRoles(memberRoles, wanted []string) bool {
	for _, w := range wanted {
		found := false
		for _, r := range memberRoles {
			if r == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
