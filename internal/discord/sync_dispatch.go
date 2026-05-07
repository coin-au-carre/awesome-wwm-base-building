package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	githubRepo      = "coin-au-carre/awesome-wwm-base-building"
	syncCooldown    = 5 * time.Minute
)

var (
	syncMu       sync.Mutex
	lastSyncTime time.Time
)

func handleSyncDataCommand(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, notifyChannelID string, allowedRoleIDs []string, githubToken string) {
	if !memberHasAnyRole(i, allowedRoleIDs) {
		respondEphemeral(s, i, "*(you need the Trusted Eye role to trigger a sync.)*")
		return
	}

	syncMu.Lock()
	elapsed := time.Since(lastSyncTime)
	if elapsed < syncCooldown {
		remaining := syncCooldown - elapsed
		syncMu.Unlock()
		respondEphemeral(s, i, fmt.Sprintf("*(a sync was just triggered — please wait %s before triggering another.)*", remaining.Round(time.Second)))
		return
	}
	syncMu.Unlock()

	active, err := workflowIsActive(githubToken)
	if err != nil {
		slog.Warn("checking workflow status", "err", err)
		// non-fatal: proceed anyway
	}
	if active {
		respondEphemeral(s, i, "*(a sync is already running or queued on GitHub — it will finish on its own.)*")
		return
	}

	slog.Info("/sync-data triggered", "user", memberDisplayName(i))

	if err := triggerGitHubWorkflow(githubToken); err != nil {
		slog.Error("triggering github workflow", "err", err)
		respondEphemeral(s, i, "*(something went wrong triggering the sync — check the logs.)*")
		return
	}

	syncMu.Lock()
	lastSyncTime = time.Now()
	syncMu.Unlock()

	respondEphemeral(s, i, "Sync triggered! Check [GitHub Actions](https://github.com/"+githubRepo+"/actions) for progress.")
	if notifyChannelID != "" {
		bot.Send(notifyChannelID, fmt.Sprintf("🔄 **%s** triggered a guild data sync.", memberDisplayName(i)))
	}
}

func workflowIsActive(token string) (bool, error) {
	type runsResponse struct {
		TotalCount int `json:"total_count"`
	}

	for _, status := range []string{"in_progress", "queued", "requested"} {
		url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/sync.yml/runs?status=%s&per_page=1", githubRepo, status)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return false, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("http: %w", err)
		}
		var result runsResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return false, fmt.Errorf("decode: %w", err)
		}
		if result.TotalCount > 0 {
			return true, nil
		}
	}
	return false, nil
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

func memberHasAnyRole(i *discordgo.InteractionCreate, roleIDs []string) bool {
	if i.Member == nil {
		return false
	}
	allowed := make(map[string]bool, len(roleIDs))
	for _, id := range roleIDs {
		if id != "" {
			allowed[id] = true
		}
	}
	for _, r := range i.Member.Roles {
		if allowed[r] {
			return true
		}
	}
	return false
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
