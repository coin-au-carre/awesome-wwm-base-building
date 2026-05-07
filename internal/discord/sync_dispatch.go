package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

const githubRepo = "coin-au-carre/awesome-wwm-base-building"

func handleSyncDataCommand(s *discordgo.Session, i *discordgo.InteractionCreate, trustedEyeRoleID, githubToken string) {
	if !memberHasRole(i, trustedEyeRoleID) {
		respondEphemeral(s, i, "*(you need the Trusted Eye role to trigger a sync.)*")
		return
	}

	slog.Info("/sync-data triggered", "user", memberDisplayName(i))

	if err := triggerGitHubWorkflow(githubToken); err != nil {
		slog.Error("triggering github workflow", "err", err)
		respondEphemeral(s, i, "*(something went wrong triggering the sync — check the logs.)*")
		return
	}

	respondEphemeral(s, i, "Sync triggered! Check [GitHub Actions](https://github.com/"+githubRepo+"/actions) for progress.")
}

func memberHasRole(i *discordgo.InteractionCreate, roleID string) bool {
	if i.Member == nil || roleID == "" {
		return false
	}
	for _, r := range i.Member.Roles {
		if r == roleID {
			return true
		}
	}
	return false
}

func triggerGitHubWorkflow(token string) error {
	body, _ := json.Marshal(map[string]string{"ref": "main"})
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/sync.yml/dispatches", githubRepo)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
