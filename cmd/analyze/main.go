// cmd/analyze/main.go — export voter analysis to CSV and HTML
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"ruby/internal/cmdutil"
	"ruby/internal/discord"

	"github.com/joho/godotenv"
)

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory")
	out := flag.String("out", "", "output directory (default: <root>/private)")
	flag.Parse()

	if err := godotenv.Load(filepath.Join(*root, ".env")); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	token := cmdutil.RequireEnv("RUBY_BOT_TOKEN")
	guildForumID := cmdutil.RequireEnv("GUILD_BASE_SHOWCASE_CHANNEL_FORUM_ID")

	outDir := *out
	if outDir == "" {
		outDir = filepath.Join(*root, "private")
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		slog.Error("creating output directory", "err", err)
		os.Exit(1)
	}

	bot, err := discord.NewBot(token, "")
	if err != nil {
		slog.Error("creating bot", "err", err)
		os.Exit(1)
	}
	defer bot.Close()
	if err := bot.Open(); err != nil {
		slog.Error("opening session", "err", err)
		os.Exit(1)
	}

	records, err := discord.AnalyzeVoters(bot, guildForumID)
	if err != nil {
		slog.Error("analyzing voters", "err", err)
		os.Exit(1)
	}
	slog.Info("records collected", "count", len(records))

	// Sort: guild name, then username
	sort.Slice(records, func(i, j int) bool {
		if records[i].GuildName != records[j].GuildName {
			return records[i].GuildName < records[j].GuildName
		}
		return records[i].Username < records[j].Username
	})

	csvPath := filepath.Join(outDir, "voter_analysis.csv")
	htmlPath := filepath.Join(outDir, "voter_analysis.html")

	if err := writeCSV(csvPath, records); err != nil {
		slog.Error("writing CSV", "err", err)
		os.Exit(1)
	}
	slog.Info("CSV written", "path", csvPath)

	if err := writeHTML(htmlPath, records); err != nil {
		slog.Error("writing HTML", "err", err)
		os.Exit(1)
	}
	slog.Info("HTML written", "path", htmlPath)
}

func writeCSV(path string, records []discord.VoteRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{
		"Guild Name", "Discord Username", "Discord ID",
		"Emojis", "Threads Voted", "Vote Weight", "Points Contributed",
	})
	for _, r := range records {
		emojis := strings.Join(r.Emojis, " ")
		_ = w.Write([]string{
			r.GuildName,
			r.Username,
			r.UserID,
			emojis,
			fmt.Sprint(r.ThreadsVoted),
			fmt.Sprint(r.Weight),
			fmt.Sprint(r.Points),
		})
	}
	return w.Error()
}

func writeHTML(path string, records []discord.VoteRecord) error {
	// Aggregate stats
	totalVoters := make(map[string]bool)
	weightDist := make(map[int]int)
	for _, r := range records {
		totalVoters[r.UserID] = true
		weightDist[r.Weight]++
	}

	guildVoterCounts := make(map[string]map[string]bool)
	for _, r := range records {
		if guildVoterCounts[r.GuildName] == nil {
			guildVoterCounts[r.GuildName] = make(map[string]bool)
		}
		guildVoterCounts[r.GuildName][r.UserID] = true
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := func(s string) { fmt.Fprint(f, s) }
	wl := func(s string) { fmt.Fprintln(f, s) }

	wl(`<!DOCTYPE html>`)
	wl(`<html lang="en">`)
	wl(`<head>`)
	wl(`<meta charset="UTF-8">`)
	wl(`<meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	wl(`<title>Voter Analysis — Awesome WWM</title>`)
	wl(`<style>`)
	wl(cssStyles())
	wl(`</style>`)
	wl(`</head>`)
	wl(`<body>`)
	wl(`<div class="container">`)
	wl(`<h1>Voter Analysis</h1>`)

	// Summary cards
	wl(`<div class="stats">`)
	wl(statCard("Total Votes", fmt.Sprint(len(records))))
	wl(statCard("Unique Voters", fmt.Sprint(len(totalVoters))))
	wl(statCard("Guilds Voted On", fmt.Sprint(len(guildVoterCounts))))
	wl(statCard("Weight×3 Voters", fmt.Sprint(weightDist[3])))
	wl(statCard("Weight×2 Voters", fmt.Sprint(weightDist[2])))
	wl(statCard("Weight×1 Voters", fmt.Sprint(weightDist[1])))
	wl(statCard("Weight×0 (no power)", fmt.Sprint(weightDist[0])))
	wl(`</div>`)

	// Filter controls
	wl(`<div class="filters">`)
	wl(`<input type="text" id="filterGuild" placeholder="Filter by guild…" oninput="applyFilters()">`)
	wl(`<input type="text" id="filterUser" placeholder="Filter by username…" oninput="applyFilters()">`)
	wl(`<select id="filterWeight" onchange="applyFilters()">`)
	wl(`  <option value="">All weights</option>`)
	wl(`  <option value="3">Weight 3</option>`)
	wl(`  <option value="2">Weight 2</option>`)
	wl(`  <option value="1">Weight 1</option>`)
	wl(`  <option value="0">Weight 0 (no power)</option>`)
	wl(`</select>`)
	wl(`<span id="rowCount"></span>`)
	wl(`</div>`)

	// Table
	wl(`<table id="voteTable">`)
	wl(`<thead><tr>`)
	for i, col := range []string{"Guild Name", "Username", "Emojis", "Threads Voted", "Weight", "Points"} {
		fmt.Fprintf(f, `<th onclick="sortTable(%d)">%s <span class="sort-icon">↕</span></th>`, i, col)
	}
	wl(`</tr></thead>`)
	wl(`<tbody>`)

	for _, r := range records {
		weightClass := fmt.Sprintf("w%d", r.Weight)
		emojiLabels := make([]string, len(r.Emojis))
		for i, e := range r.Emojis {
			emojiLabels[i] = discord.EmojiLabel(e)
		}
		slices.Sort(emojiLabels)
		emojiStr := strings.Join(emojiLabels, ", ")

		fmt.Fprintf(f, `<tr data-guild="%s" data-user="%s" data-weight="%d">`,
			html.EscapeString(strings.ToLower(r.GuildName)),
			html.EscapeString(strings.ToLower(r.Username)),
			r.Weight,
		)
		fmt.Fprintf(f, `<td>%s</td>`, html.EscapeString(r.GuildName))
		fmt.Fprintf(f, `<td>%s</td>`, html.EscapeString(r.Username))
		fmt.Fprintf(f, `<td>%s</td>`, html.EscapeString(emojiStr))
		fmt.Fprintf(f, `<td class="num">%d</td>`, r.ThreadsVoted)
		fmt.Fprintf(f, `<td class="num %s">%d</td>`, weightClass, r.Weight)
		fmt.Fprintf(f, `<td class="num">%d</td>`, r.Points)
		wl(`</tr>`)
	}

	wl(`</tbody>`)
	wl(`</table>`)

	// Embed JS
	w(`<script>`)
	wl(jsScript())
	wl(`</script>`)

	wl(`</div>`)
	wl(`</body>`)
	wl(`</html>`)
	return nil
}

func statCard(label, value string) string {
	return fmt.Sprintf(`<div class="card"><div class="card-value">%s</div><div class="card-label">%s</div></div>`,
		html.EscapeString(value), html.EscapeString(label))
}

func cssStyles() string {
	return `
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, sans-serif; background: #0f1117; color: #e2e8f0; }
.container { max-width: 1400px; margin: 0 auto; padding: 24px; }
h1 { font-size: 1.75rem; margin-bottom: 20px; color: #f8fafc; }

.stats { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 24px; }
.card { background: #1e2330; border: 1px solid #2d3447; border-radius: 8px; padding: 14px 18px; min-width: 120px; }
.card-value { font-size: 1.6rem; font-weight: 700; color: #60a5fa; }
.card-label { font-size: 0.75rem; color: #94a3b8; margin-top: 2px; }

.filters { display: flex; gap: 10px; align-items: center; margin-bottom: 16px; flex-wrap: wrap; }
.filters input, .filters select {
  background: #1e2330; border: 1px solid #2d3447; color: #e2e8f0;
  border-radius: 6px; padding: 7px 12px; font-size: 0.875rem; outline: none;
}
.filters input:focus, .filters select:focus { border-color: #60a5fa; }
#rowCount { font-size: 0.8rem; color: #64748b; }

table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
thead { position: sticky; top: 0; background: #1a1f2e; z-index: 10; }
th {
  padding: 10px 12px; text-align: left; color: #94a3b8;
  font-weight: 600; cursor: pointer; user-select: none; white-space: nowrap;
  border-bottom: 1px solid #2d3447;
}
th:hover { color: #e2e8f0; }
.sort-icon { opacity: 0.4; font-size: 0.7rem; }
td { padding: 8px 12px; border-bottom: 1px solid #1e2330; }
tr:hover td { background: #1e2330; }
tr.hidden { display: none; }
.num { text-align: right; font-variant-numeric: tabular-nums; }

.w3 { color: #fbbf24; font-weight: 700; }
.w2 { color: #60a5fa; font-weight: 600; }
.w1 { color: #94a3b8; }
.w0 { color: #4b5563; }
`
}

func jsScript() string {
	return `
var sortDir = {};
function sortTable(col) {
  var tbody = document.querySelector('#voteTable tbody');
  var rows = Array.from(tbody.querySelectorAll('tr'));
  var asc = !sortDir[col];
  sortDir = {};
  sortDir[col] = asc;
  rows.sort(function(a, b) {
    var av = a.cells[col].textContent.trim();
    var bv = b.cells[col].textContent.trim();
    var an = parseFloat(av), bn = parseFloat(bv);
    if (!isNaN(an) && !isNaN(bn)) return asc ? an - bn : bn - an;
    return asc ? av.localeCompare(bv) : bv.localeCompare(av);
  });
  rows.forEach(function(r) { tbody.appendChild(r); });
  document.querySelectorAll('.sort-icon').forEach(function(el) { el.textContent = '↕'; });
  var th = document.querySelectorAll('th')[col];
  th.querySelector('.sort-icon').textContent = asc ? '↑' : '↓';
  applyFilters();
}
function applyFilters() {
  var guild = document.getElementById('filterGuild').value.toLowerCase();
  var user = document.getElementById('filterUser').value.toLowerCase();
  var weight = document.getElementById('filterWeight').value;
  var rows = document.querySelectorAll('#voteTable tbody tr');
  var visible = 0;
  rows.forEach(function(r) {
    var show = (!guild || r.dataset.guild.includes(guild))
            && (!user  || r.dataset.user.includes(user))
            && (!weight || r.dataset.weight === weight);
    r.classList.toggle('hidden', !show);
    if (show) visible++;
  });
  document.getElementById('rowCount').textContent = visible + ' rows';
}
window.addEventListener('DOMContentLoaded', function() {
  applyFilters();
});
`
}
