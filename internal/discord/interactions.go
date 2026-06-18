package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const (
	submitCommandName      = "scout-guild"
	submitModalID          = "scout_guild_modal"
	postCommandName        = "submit-guild"
	postModalID            = "submit_guild_modal"
	soloCommandName        = "submit-solo"
	soloModalID            = "submit_solo_modal"
	welcomeTestCommandName = "welcome-test"
	guildLinkCommandName   = "guild"
	soloLinkCommandName    = "solo"
	randomCommandName      = "random"
	myVotesCommandName     = "my-votes"
	helpCommandName        = "commands"
	builderCommandName     = "builder"
	warningListCommandName = "warning-list"
	syncDataCommandName    = "sync-data"
	syncBugsCommandName    = "sync-bugs"
	syncUpdatesCommandName = "sync-updates"
	syncTagsCommandName    = "sync-tags"
	shareGuildPrefix       = "sbg:"
	shareSoloPrefix        = "sbs:"
	dotifyCommandName      = "dotify"
)

var submitMu sync.Mutex

func logCommandUsage(bot *Bot, i *discordgo.InteractionCreate, devChannelID string) {
	if devChannelID == "" {
		return
	}
	name := i.ApplicationCommandData().Name
	var user string
	if i.Member != nil {
		user = i.Member.DisplayName()
	} else if i.User != nil {
		user = i.User.Username
	}
	var opts []string
	for _, o := range i.ApplicationCommandData().Options {
		opts = append(opts, o.Name+":"+fmt.Sprint(o.Value))
	}
	msg := fmt.Sprintf("`/%s` used by **%s**", name, user)
	if len(opts) > 0 {
		msg += " — " + strings.Join(opts, ", ")
	}
	bot.Send(devChannelID, msg)
}

func RegisterSubmitCommand(s *discordgo.Session, discordGuildID string) {
	adminPerm := int64(discordgo.PermissionAdministrator)

	// Global commands — visible on Ruby's profile card, propagate within ~1 hour.
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", []*discordgo.ApplicationCommand{
		{
			Name:        submitCommandName,
			Description: "Scout a guild base and reference it for later",
		},
		{
			Name:        postCommandName,
			Description: "Submit your guild base to showcase",
		},
		{
			Name:        soloCommandName,
			Description: "Submit your solo construction to showcase",
		},
		{
			Name:        guildLinkCommandName,
			Description: "Share a guild base with its thread and page",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "name",
					Description:  "Guild name",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        soloLinkCommandName,
			Description: "Share a solo base with its thread and page",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "name",
					Description:  "Solo build name",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        randomCommandName,
			Description: "Ruby picks a random guild base to showcase",
		},
		{
			Name:        myVotesCommandName,
			Description: "Show the guild bases you have voted for (only you see the result)",
		},
		{
			Name:        helpCommandName,
			Description: "List all Ruby commands (only you see the result)",
		},
		{
			Name:        builderCommandName,
			Description: "Look up a member's guild or solo base (only you see the result)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "member",
					Description: "Server member",
					Required:    true,
				},
			},
		},
		{
			Name:        dotifyCommandName,
			Description: "Replace dots in a URL with ｡ so it can be shared in Where Builders Meet",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "URL to transform",
					Required:    true,
				},
			},
		},
	})
	if err != nil {
		slog.Error("registering global commands", "err", err)
	}

	// Guild-only commands — mod/admin tools, instant propagation.
	_, err = s.ApplicationCommandBulkOverwrite(s.State.User.ID, discordGuildID, []*discordgo.ApplicationCommand{
		{
			Name:                     welcomeTestCommandName,
			Description:              "Preview the welcome message (admins only)",
			DefaultMemberPermissions: &adminPerm,
		},
		{
			Name:        warningListCommandName,
			Description: "Show the warning list (only you see the result)",
		},
		{
			Name:        syncDataCommandName,
			Description: "Trigger data sync on guilds, solo, tutorials (elevated roles only)",
		},
		{
			Name:        syncBugsCommandName,
			Description: "Trigger bug list sync from the Super Sheet (elevated roles only)",
		},
		{
			Name:        syncUpdatesCommandName,
			Description: "Trigger patch notes sync from the Super Sheet (elevated roles only)",
		},
		{
			Name:                     syncTagsCommandName,
			Description:              "Sync forum channel tags to the canonical list in config/tags.json (admins only)",
			DefaultMemberPermissions: &adminPerm,
		},
	})
	if err != nil {
		slog.Error("registering guild commands", "err", err)
	}
}

func OnInteractionCreate(bot *Bot, root, submissionChannelID, discoveriesChannelID, guildForumChannelID, soloForumChannelID, devChannelID, botChannelID, trustedEyeRoleID, trustedMemberRoleID, githubToken string, responder LLMResponder) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommandAutocomplete:
			switch i.ApplicationCommandData().Name {
			case guildLinkCommandName:
				handleGuildLinkAutocomplete(s, i, root)
			case soloLinkCommandName:
				handleSoloLinkAutocomplete(s, i, root)
			}
		case discordgo.InteractionApplicationCommand:
			logCommandUsage(bot, i, devChannelID)
			switch i.ApplicationCommandData().Name {
			case submitCommandName:
				handleSubmitCommand(s, i, root)
			case postCommandName:
				handlePostCommand(s, i)
			case soloCommandName:
				handleSoloCommand(s, i)
			case welcomeTestCommandName:
				handleWelcomeTestCommand(s, i)
			case guildLinkCommandName:
				handleGuildLinkCommand(s, i, root)
			case soloLinkCommandName:
				handleSoloLinkCommand(s, i, root)
			case randomCommandName:
				handleRandomCommand(s, i, root, responder)
			case myVotesCommandName:
				handleMyVotesCommand(s, i, root)
			case helpCommandName:
				handleHelpCommand(s, i)
			case builderCommandName:
				handleBuilderCommand(s, i, root)
			case warningListCommandName:
				handleWarningListCommand(s, i)
			case syncDataCommandName:
				handleSyncDataCommand(s, i, bot, botChannelID, []string{trustedEyeRoleID, trustedMemberRoleID}, githubToken)
			case syncBugsCommandName:
				handleSyncBugsCommand(s, i, bot, botChannelID, []string{trustedEyeRoleID, trustedMemberRoleID}, githubToken)
			case syncUpdatesCommandName:
				handleSyncUpdatesCommand(s, i, bot, botChannelID, []string{trustedEyeRoleID, trustedMemberRoleID}, githubToken)
			case syncTagsCommandName:
				handleSyncTagsCommand(s, i, root, guildForumChannelID, soloForumChannelID)
			case dotifyCommandName:
				handleDotifyCommand(s, i)
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			if strings.HasPrefix(customID, shareGuildPrefix) || strings.HasPrefix(customID, shareSoloPrefix) {
				handleShareBuilderButton(s, i, root)
			}
		case discordgo.InteractionModalSubmit:
			switch i.ModalSubmitData().CustomID {
			case submitModalID:
				handleSubmitModal(s, i, bot, root, submissionChannelID, discoveriesChannelID, devChannelID)
			case postModalID:
				handlePostModal(s, i, bot, devChannelID, guildForumChannelID)
			case soloModalID:
				handleSoloModal(s, i, bot, submissionChannelID, soloForumChannelID, devChannelID)
			}
		}
	}
}
