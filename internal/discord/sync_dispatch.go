package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	githubRepo   = "coin-au-carre/awesome-wwm-base-building"
	syncCooldown = 5 * time.Minute
	pollInterval = 10 * time.Second
	pollTimeout  = 10 * time.Minute
	findRunLimit = 30 * time.Second
)

type syncState struct {
	mu       sync.Mutex
	lastTime time.Time
}

var (
	guildSync   syncState
	bugsSync    syncState
	updatesSync syncState
)

func handleSyncDataCommand(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, notifyChannelID string, allowedRoleIDs []string, githubToken string) {
	handleSyncCommand(s, i, bot, notifyChannelID, allowedRoleIDs, githubToken, "sync.yml", "Guild data", &guildSync)
}

func handleSyncBugsCommand(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, notifyChannelID string, allowedRoleIDs []string, githubToken string) {
	handleSyncCommand(s, i, bot, notifyChannelID, allowedRoleIDs, githubToken, "bugs-sync.yml", "Bug data", &bugsSync)
}

func handleSyncUpdatesCommand(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, notifyChannelID string, allowedRoleIDs []string, githubToken string) {
	handleSyncCommand(s, i, bot, notifyChannelID, allowedRoleIDs, githubToken, "patches-sync.yml", "Updates data", &updatesSync)
}

func handleSyncCommand(s *discordgo.Session, i *discordgo.InteractionCreate, bot *Bot, notifyChannelID string, allowedRoleIDs []string, githubToken, workflow, label string, state *syncState) {
	if !memberHasAnyRole(i, allowedRoleIDs) {
		respondEphemeral(s, i, "*(you need the Trusted Eye or Trusted Member role to trigger a sync.)*")
		return
	}

	state.mu.Lock()
	elapsed := time.Since(state.lastTime)
	if elapsed < syncCooldown {
		remaining := syncCooldown - elapsed
		state.mu.Unlock()
		respondEphemeral(s, i, fmt.Sprintf("*(a sync was just triggered — please wait %s before triggering another.)*", remaining.Round(time.Second)))
		return
	}
	state.mu.Unlock()

	active, err := workflowIsActive(githubToken, workflow)
	if err != nil {
		slog.Warn("checking workflow status", "err", err)
	}
	if active {
		respondEphemeral(s, i, "*(a sync is already running or queued — it will finish on its own.)*")
		return
	}

	triggerTime := time.Now()
	if err := triggerGitHubWorkflow(githubToken, workflow); err != nil {
		slog.Error("triggering github workflow", "err", err)
		respondEphemeral(s, i, "*(something went wrong triggering the sync. Ask Ahlyam.)*")
		return
	}

	state.mu.Lock()
	state.lastTime = triggerTime
	state.mu.Unlock()

	name := memberDisplayName(i)
	slog.Info("sync triggered", "workflow", workflow, "user", name)
	respondEphemeral(s, i, fmt.Sprintf("Sync triggered! Progress will appear in <#%s>.", notifyChannelID))

	if notifyChannelID != "" {
		header := fmt.Sprintf("🔄 %s sync manually triggered by **%s**", label, name)
		msgID := bot.SendReturnID(notifyChannelID, header+" — looking for run...")
		if msgID != "" {
			go pollSyncProgress(bot, notifyChannelID, msgID, header, triggerTime, githubToken, workflow)
		}
	}
}

func pollSyncProgress(bot *Bot, channelID, msgID, header string, triggerTime time.Time, token, workflow string) {
	// Wait for the run to appear on GitHub.
	var runID int64
	deadline := time.Now().Add(findRunLimit)
	for time.Now().Before(deadline) {
		id, found, err := findWorkflowRun(token, triggerTime, workflow)
		if err != nil {
			slog.Warn("finding workflow run", "err", err)
		}
		if found {
			runID = id
			break
		}
		time.Sleep(5 * time.Second)
	}
	if runID == 0 {
		bot.EditMessage(channelID, msgID, header+" — *(could not find the run on GitHub.)*")
		return
	}

	start := time.Now()
	timeout := time.After(pollTimeout)
	for {
		select {
		case <-timeout:
			bot.EditMessage(channelID, msgID, header+" — *(timed out waiting for completion.)*")
			return
		default:
		}

		prog, err := fetchRunProgress(token, runID)
		if err != nil {
			slog.Warn("fetching run progress", "err", err)
			time.Sleep(pollInterval)
			continue
		}

		bar := ProgressBar(prog.pct)
		elapsed := time.Since(start).Round(time.Second)

		if prog.done {
			var icon, label string
			switch prog.conclusion {
			case "success":
				icon, label = "✅", "completed"
			case "cancelled":
				icon, label = "🚫", "cancelled"
			default:
				icon, label = "❌", "failed"
			}
			bot.EditMessage(channelID, msgID, fmt.Sprintf(
				"%s %s — %s in %s\n%s 100%%",
				icon, header[2:], label, elapsed, bar,
			))
			return
		}

		stepInfo := ""
		if prog.currentStep != "" {
			stepInfo = " • " + rubyStepName(prog.currentStep)
		}
		bot.EditMessage(channelID, msgID, fmt.Sprintf(
			"%s\n%s %d%%%s — _%s elapsed_",
			header, bar, prog.pct, stepInfo, elapsed,
		))

		time.Sleep(pollInterval)
	}
}

func rubyStepName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "set up job"):
		return "*(stretching awake...)*"
	case strings.Contains(lower, "checkout"):
		return "*(finding the path...)*"
	case strings.Contains(lower, "go"):
		return "*(sharpening the tools...)*"
	case strings.Contains(lower, "task"):
		return "*(reading the scrolls...)*"
	case strings.Contains(lower, "sync"):
		return "*(wandering through guild halls...)*"
	case strings.Contains(lower, "commit"):
		return "*(sealing the records...)*"
	case strings.Contains(lower, "post"):
		return "*(tidying up...)*"
	default:
		return name
	}
}

func ProgressBar(pct int) string {
	if pct > 100 {
		pct = 100
	}
	filled := pct / 10
	return strings.Repeat("▓", filled) + strings.Repeat("░", 10-filled)
}

type runProgress struct {
	pct         int
	currentStep string
	done        bool
	conclusion  string
}

func findWorkflowRun(token string, after time.Time, workflow string) (int64, bool, error) {
	type run struct {
		ID        int64  `json:"id"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
	type response struct {
		WorkflowRuns []run `json:"workflow_runs"`
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/runs?per_page=5&event=workflow_dispatch", githubRepo, workflow)
	req, err := githubRequest(http.MethodGet, url, token, nil)
	if err != nil {
		return 0, false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, false, fmt.Errorf("decode: %w", err)
	}

	for _, r := range result.WorkflowRuns {
		t, err := time.Parse(time.RFC3339, r.CreatedAt)
		if err != nil {
			continue
		}
		if !t.Before(after.Add(-10 * time.Second)) {
			return r.ID, true, nil
		}
	}
	return 0, false, nil
}

func fetchRunProgress(token string, runID int64) (runProgress, error) {
	type step struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	type job struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		Steps      []step `json:"steps"`
	}
	type runResp struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	}
	type jobsResp struct {
		Jobs []job `json:"jobs"`
	}

	// Check overall run status first.
	runURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d", githubRepo, runID)
	req, err := githubRequest(http.MethodGet, runURL, token, nil)
	if err != nil {
		return runProgress{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return runProgress{}, fmt.Errorf("http: %w", err)
	}
	var run runResp
	err = json.NewDecoder(resp.Body).Decode(&run)
	resp.Body.Close()
	if err != nil {
		return runProgress{}, fmt.Errorf("decode run: %w", err)
	}

	if run.Status == "completed" {
		return runProgress{done: true, conclusion: run.Conclusion, pct: 100}, nil
	}

	// Fetch jobs + steps for progress.
	jobsURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/jobs", githubRepo, runID)
	req, err = githubRequest(http.MethodGet, jobsURL, token, nil)
	if err != nil {
		return runProgress{}, err
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return runProgress{}, fmt.Errorf("http jobs: %w", err)
	}
	var jobs jobsResp
	err = json.NewDecoder(resp.Body).Decode(&jobs)
	resp.Body.Close()
	if err != nil {
		return runProgress{}, fmt.Errorf("decode jobs: %w", err)
	}

	var total, completed int
	var current string
	for _, j := range jobs.Jobs {
		for _, st := range j.Steps {
			total++
			switch st.Status {
			case "completed":
				completed++
			case "in_progress":
				current = st.Name
			}
		}
	}

	pct := 0
	if total > 0 {
		pct = completed * 100 / total
	}
	return runProgress{pct: pct, currentStep: current}, nil
}

func workflowIsActive(token, workflow string) (bool, error) {
	type runsResponse struct {
		TotalCount int `json:"total_count"`
	}

	for _, status := range []string{"in_progress", "queued", "requested"} {
		url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/runs?status=%s&per_page=1", githubRepo, workflow, status)
		req, err := githubRequest(http.MethodGet, url, token, nil)
		if err != nil {
			return false, err
		}
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

func triggerGitHubWorkflow(token, workflow string) error {
	dispatchURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches", githubRepo, workflow)

	payload, _ := json.Marshal(map[string]any{
		"ref": "main",
	})
	req, err := githubRequest(http.MethodPost, dispatchURL, token, payload)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		var gh struct {
			Message string `json:"message"`
		}
		if jsonErr := json.NewDecoder(resp.Body).Decode(&gh); jsonErr == nil && gh.Message != "" {
			return fmt.Errorf("status %d: %s", resp.StatusCode, gh.Message)
		}
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func githubRequest(method, url, token string, body []byte) (*http.Request, error) {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return req, nil
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
