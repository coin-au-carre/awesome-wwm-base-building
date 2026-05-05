package discord

import (
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
	myVotesCommandName      = "my-votes"
	helpCommandName         = "commands"
	builderCommandName      = "builder"
	warningListCommandName  = "warning-list"
	shareGuildPrefix       = "sbg:"
	shareSoloPrefix        = "sbs:"
)

var submitMu sync.Mutex

func RegisterSubmitCommand(s *discordgo.Session, discordGuildID string) {
	adminPerm := int64(discordgo.PermissionAdministrator)
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, discordGuildID, []*discordgo.ApplicationCommand{
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
			Name:                     welcomeTestCommandName,
			Description:              "Preview the welcome message (admins only)",
			DefaultMemberPermissions: &adminPerm,
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
			Name:        warningListCommandName,
			Description: "Show the warning list (only you see the result)",
		},
	})
	if err != nil {
		slog.Error("registering commands", "err", err)
	}
}

func OnInteractionCreate(bot *Bot, root, submissionChannelID, discoveriesChannelID, guildForumChannelID, soloForumChannelID, devChannelID string, responder LLMResponder) func(*discordgo.Session, *discordgo.InteractionCreate) {
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
			switch i.ApplicationCommandData().Name {
			case submitCommandName:
				handleSubmitCommand(s, i)
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
			}
		case discordgo.InteractionMessageComponent:
			customID := i.MessageComponentData().CustomID
			if strings.HasPrefix(customID, shareGuildPrefix) || strings.HasPrefix(customID, shareSoloPrefix) {
				handleShareBuilderButton(s, i, root)
			}
		case discordgo.InteractionModalSubmit:
			switch i.ModalSubmitData().CustomID {
			case submitModalID:
				handleSubmitModal(s, i, bot, root, submissionChannelID, discoveriesChannelID)
			case postModalID:
				handlePostModal(s, i, bot, devChannelID, guildForumChannelID)
			case soloModalID:
				handleSoloModal(s, i, bot, submissionChannelID, soloForumChannelID)
			}
		}
	}
}
