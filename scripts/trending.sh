#!/usr/bin/env bash
# Rising-fastest / newest guilds.
# Usage:
#   scripts/trending.sh [days] [top_n]        rising fastest: score delta over the last N days (git history)
#   scripts/trending.sh [days] [top_n] new    new guilds: createdAt within the last N days, sorted by score
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

DAYS="${1:-30}"
TOP_N="${2:-15}"
MODE="${3:-score}"
SINCE=$(date -d "-${DAYS} days" +%Y-%m-%d)

if [ "$MODE" = "new" ]; then
  jq -rn --slurpfile new data/guilds.json --argjson n "$TOP_N" --arg since "$SINCE" '
    ($since | strptime("%Y-%m-%d") | mktime) as $cutoff |
    $new[0]
    | map(select(.createdAt != null))
    | map(. + {createdMs: (.createdAt | strptime("%B %d, %Y at %I:%M %p UTC") | mktime)})
    | map(select(.createdMs >= $cutoff))
    | sort_by(-.score)
    | .[:$n]
    | .[]
    | "\(.score)\t\(.createdAt)\t\(.guildName // .name)"
  ' | column -t -s $'\t' -N "SCORE,CREATED,GUILD"
  exit 0
fi

OLD_COMMIT=$(git log --format="%H %ad" --date=short -- data/guilds.json | awk -v since="$SINCE" '$2<=since{print $1; exit}'; true)
if [ -z "$OLD_COMMIT" ]; then
  echo "No commit found on or before $SINCE" >&2
  exit 1
fi

OLD_JSON=$(mktemp)
trap 'rm -f "$OLD_JSON"' EXIT
git show "${OLD_COMMIT}:data/guilds.json" > "$OLD_JSON"

jq -rn --slurpfile old "$OLD_JSON" --slurpfile new data/guilds.json --argjson n "$TOP_N" '
  ($old[0] | map({(.discordThread // .name): .score}) | add // {}) as $oldScores |
  $new[0]
  | map({name: (.guildName // .name), thread: .discordThread, score, delta: (.score - ($oldScores[.discordThread // .name] // .score))})
  | map(select(.delta > 0))
  | sort_by(-.delta)
  | .[:$n]
  | .[]
  | "+\(.delta)\t\(.score)\t\(.name)"
' | column -t -s $'\t' -N "DELTA,SCORE,GUILD"
