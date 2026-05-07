package discord

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const dataSyncInterval = 5 * time.Minute

type reloader interface {
	Reload(root string)
}

var (
	pullLineMu   sync.Mutex
	pullLineOpen bool // true while a pull log line has no trailing newline yet
)

// logPull appends to an open pull line (as " #") or starts a new one.
// Callers must not hold pullLineMu.
func logPull(files string) {
	pullLineMu.Lock()
	defer pullLineMu.Unlock()
	if pullLineOpen {
		fmt.Fprint(os.Stderr, " #")
	} else {
		fmt.Fprintf(os.Stderr, "%s INFO data watcher: pulled (%s) #",
			time.Now().Format("2006/01/02 15:04:05"), files)
		pullLineOpen = true
	}
}

// closePullLine ends the open pull line with a newline, if any.
func closePullLine() {
	pullLineMu.Lock()
	defer pullLineMu.Unlock()
	if pullLineOpen {
		fmt.Fprintln(os.Stderr)
		pullLineOpen = false
	}
}

// PullOnStart runs a git pull --rebase at startup and reloads the responder
// if data files changed. Errors are logged but do not stop the bot.
func PullOnStart(root string, responder LLMResponder) {
	r, ok := responder.(reloader)
	if !ok {
		return
	}
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	before, err := git("rev-parse", "HEAD:data/guilds.json")
	if err != nil {
		slog.Warn("startup pull: could not read current data hash", "err", err)
	}

	if out, err := git("pull", "--rebase"); err != nil {
		slog.Warn("startup pull: git pull failed", "err", err, "output", out)
		_, _ = git("rebase", "--abort")
		return
	}

	after, _ := git("rev-parse", "HEAD:data/guilds.json")
	if after != before {
		r.Reload(root)
		slog.Info("startup pull: data updated and reloaded")
	} else {
		slog.Info("startup pull: already up to date")
	}
}

// StartDataWatcher polls git every 5 minutes, and when data/guilds.json or
// data/solos.json have changed on origin/main it pulls and reloads the responder.
// It returns immediately; the polling runs in a background goroutine until ctx is done.
func StartDataWatcher(ctx context.Context, root string, responder LLMResponder) {
	r, ok := responder.(reloader)
	if !ok {
		return
	}
	go func() {
		ticker := time.NewTicker(dataSyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				closePullLine()
				return
			case <-ticker.C:
				pullAndReloadIfChanged(root, r)
			}
		}
	}()
}

func pullAndReloadIfChanged(root string, r reloader) {
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	if _, err := git("fetch"); err != nil {
		closePullLine()
		slog.Warn("data watcher: git fetch failed", "err", err)
		return
	}

	diff, err := git("diff", "--name-only", "HEAD", "origin/main", "--", "data/guilds.json", "data/solos.json")
	if err != nil {
		closePullLine()
		slog.Warn("data watcher: git diff failed", "err", err)
		return
	}
	if diff == "" {
		return
	}

	if out, err := git("pull", "--rebase"); err != nil {
		closePullLine()
		slog.Warn("data watcher: git pull failed", "err", err, "output", out)
		_, _ = git("rebase", "--abort")
		return
	}

	r.Reload(root)
	logPull(diff)
}
