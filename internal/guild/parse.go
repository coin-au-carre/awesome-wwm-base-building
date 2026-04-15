package guild

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reThreadID        = regexp.MustCompile(`\s*[\[(](\d+)[\])]?\s*$`)
	reBracketID       = regexp.MustCompile(`[\[(](\d+)[\])]`)
	reEightDigit      = regexp.MustCompile(`\b(\d{8})\b`)
	reEightDigitName  = regexp.MustCompile(`\s*[\[(]\d{8}[\])]|\s+\d{8}\b`)
	reBuilders        = regexp.MustCompile(`(?i)builders?:[ \t]*([^\n]*)`)
	reGuildName       = regexp.MustCompile(`(?m)^[#\s]*(?::[^:]+:|\*\*|\p{So}\s*)*(.+?)\**\s*[\[(]\d{6,9}[\])]`)
	reGuildNameEq     = regexp.MustCompile(`🏯[^=\n]*=\s*([^\n]+)`)
	reLore            = regexp.MustCompile(`(?im)(?:^###[^\n]*lore|\*\*\s*lore\s*\*\*|\blore\b)[^\n]*\n+([\s\S]*?)(?:\p{So}\s*)?(?:\*\*\s*what\s+to\s+visit\s*\*\*|\bwhat\s+to\s+visit\b|^###|\z)`)
	reLoreEq          = regexp.MustCompile(`(?im)\blore\b\s*=\s*([^\n]+)`)
	reWhatToVisit     = regexp.MustCompile(`(?im)(?:^###[^\n]*what\s+to\s+visit|\*\*\s*what\s+to\s+visit\s*\*\*|\bwhat\s+to\s+visit\b)[^\n]*\n+([\s\S]*?)(?:🗳|^###|\z)`)
	reWhatToVisitEq   = regexp.MustCompile(`(?im)what\s+to\s+visit\s*=\s*([^\n]+)`)
	reCover           = regexp.MustCompile(`(?i)cover:[ \t]*(\d+)`)
	reTrailingStars   = regexp.MustCompile(`(?:\s*\n\s*\*+)+\s*$`)
)

var skipPhrases = []string{
	"replace_with_your_lore",
	"describe_point_of_interest",
	"claim ownership",
	"contact us",
}

// ParseFirstPost extracts structured data from the first message of a Discord thread.
// coverIdx is 1-based; 0 means not specified.
func ParseFirstPost(content string) (id string, guildName string, builders []string, lore string, whatToVisit string, coverIdx int) {
	if m := reBracketID.FindStringSubmatch(content); len(m) > 1 {
		id = m[1]
	} else if m := reEightDigit.FindStringSubmatch(content); len(m) > 1 {
		id = m[1]
	}

	if m := reGuildName.FindStringSubmatch(content); len(m) > 1 {
		guildName = strings.TrimSpace(m[1])
	} else if m := reGuildNameEq.FindStringSubmatch(content); len(m) > 1 {
		guildName = strings.TrimSpace(m[1])
	}

	if m := reBuilders.FindStringSubmatch(content); len(m) > 1 {
		for _, b := range strings.Split(m[1], ",") {
			b = strings.TrimSpace(b)
			if b != "" && !strings.HasPrefix(b, "#") && !strings.HasPrefix(b, ":") {
				builders = append(builders, b)
			}
		}
	}

	if m := reLore.FindStringSubmatch(content); len(m) > 1 {
		lore = cleanSection(m[1])
	}
	if lore == "" {
		if m := reLoreEq.FindStringSubmatch(content); len(m) > 1 {
			lore = cleanSection(m[1])
		}
	}

	if m := reWhatToVisit.FindStringSubmatch(content); len(m) > 1 {
		whatToVisit = cleanSection(m[1])
	}
	if whatToVisit == "" {
		if m := reWhatToVisitEq.FindStringSubmatch(content); len(m) > 1 {
			whatToVisit = cleanSection(m[1])
		}
	}

	if m := reCover.FindStringSubmatch(content); len(m) > 1 {
		fmt.Sscan(m[1], &coverIdx)
	}

	return
}

// ExtractNameAndID strips decorators/suffixes from a Discord thread name and
// extracts an optional numeric guild ID embedded as a trailing bracket token.
// e.g. "WITCHERS [10248427" → ("WITCHERS", "10248427")
// e.g. "🏯 Iron Keep - Season 2" → ("Iron Keep", "")
func ExtractNameAndID(threadName string) (name, id string) {
	parts := strings.SplitN(threadName, " -", 2)
	raw := strings.TrimSpace(strings.Trim(parts[0], "🏯📍"))
	if m := reThreadID.FindStringSubmatch(raw); len(m) == 2 {
		id = m[1]
		raw = reThreadID.ReplaceAllString(raw, "")
	}
	name = strings.TrimSpace(strings.Trim(raw, "[] "))
	name = strings.TrimSpace(reEightDigitName.ReplaceAllString(name, ""))
	return
}

// ExtractName strips decorators and suffixes from a Discord thread name.
// e.g. "🏯 Iron Keep - Season 2" → "Iron Keep"
func ExtractName(threadName string) string {
	name, _ := ExtractNameAndID(threadName)
	return name
}

// IsImage reports whether filename has a recognised image extension.
func IsImage(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// IsVideo reports whether filename has a recognised video extension.
func IsVideo(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp4", ".mov", ".webm", ".mkv", ".avi":
		return true
	}
	return false
}

func cleanSection(s string) string {
	s = strings.TrimSpace(s)
	s = reTrailingStars.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if len(s) < 15 {
		return ""
	}
	lower := strings.ToLower(s)
	for _, phrase := range skipPhrases {
		if strings.Contains(lower, phrase) {
			return ""
		}
	}
	return s
}

