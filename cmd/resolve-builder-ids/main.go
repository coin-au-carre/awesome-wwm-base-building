// One-shot backfill: for every data/builder_identities.json record that
// has a neteaseNumberId but is still missing neteasePid/neteaseHostnum
// (e.g. seeded by hand, or registered before this tool existed), resolve
// it live via NetEase's find_people/by_number_id and fill it in. See
// docs/builder-identity.md's Piece 3 — /wwm-uid resolves and stores these
// at registration time going forward, so this is only needed for the
// backlog, not routine per-registration work. No credentials needed —
// these NetEase reads are confirmed unauthenticated.
package main

import (
	"log/slog"
	"os"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"
	"ruby/internal/netease"
)

func main() {
	root := cmdutil.RootDir()
	cmdutil.LoadEnv(root)

	identities, err := discord.LoadBuilderIdentities(root)
	if err != nil {
		slog.Error("loading builder_identities.json, refusing to run", "err", err)
		os.Exit(1)
	}

	resolved, skipped, failed := 0, 0, 0
	for idx, entry := range identities {
		if entry.NeteaseNumberID == "" {
			continue
		}
		if entry.NeteasePID != "" && entry.NeteaseHostnum != 0 {
			skipped++
			continue
		}

		ref, err := netease.ResolveByNumberID(entry.NeteaseNumberID)
		if err != nil {
			slog.Warn("resolving number_id failed, leaving for next run",
				"canonicalSlug", entry.CanonicalSlug, "numberId", entry.NeteaseNumberID, "err", err)
			failed++
			continue
		}

		identities[idx].NeteasePID = ref.PID
		identities[idx].NeteaseHostnum = ref.Hostnum
		identities[idx].IngameNickname = ref.Nickname
		resolved++
		slog.Info("resolved", "canonicalSlug", entry.CanonicalSlug, "numberId", entry.NeteaseNumberID, "nickname", ref.Nickname)
	}

	if resolved == 0 {
		slog.Info("nothing to resolve", "already_resolved", skipped, "failed", failed)
		return
	}

	if err := discord.SaveBuilderIdentities(root, identities); err != nil {
		slog.Error("saving builder_identities.json", "err", err)
		os.Exit(1)
	}
	slog.Info("done", "resolved", resolved, "already_resolved", skipped, "failed", failed)
}
