package guild

import (
	"regexp"
	"strings"
)

var (
	reBracketID   = regexp.MustCompile(`[\[(](\d+)[\])]`)
	reEightDigit  = regexp.MustCompile(`\b(\d{8})\b`)
	reBuilders    = regexp.MustCompile(`(?i)builders?:\s*([^\n]*)`)
	reGuildName   = regexp.MustCompile(`(?m)^[#\s]*(?::[^:]+:|\*\*|\p{So}\s*)*(.+?)\**\s*[\[(]\d{6,9}[\])]`)
	reLore        = regexp.MustCompile(`(?im)(?:^###[^\n]*lore|\*\*\s*lore\s*\*\*|\blore\b)[^\n]*\n+([\s\S]*?)(?:\p{So}\s*)?(?:\*\*\s*what\s+to\s+visit\s*\*\*|\bwhat\s+to\s+visit\b|^###|\z)`)
	reWhatToVisit = regexp.MustCompile(`(?im)(?:^###[^\n]*what\s+to\s+visit|\*\*\s*what\s+to\s+visit\s*\*\*|\bwhat\s+to\s+visit\b)[^\n]*\n+([\s\S]*?)(?:🗳|^###|\z)`)
)

var skipPhrases = []string{
	"replace_with_your_lore",
	"describe_point_of_interest",
	"claim ownership",
	"contact us",
}

// ParseFirstPost extracts structured data from the first message of a Discord thread.
func ParseFirstPost(content string) (id string, guildName string, builders []string, lore string, whatToVisit string) {
	if m := reBracketID.FindStringSubmatch(content); len(m) > 1 {
		id = m[1]
	} else if m := reEightDigit.FindStringSubmatch(content); len(m) > 1 {
		id = m[1]
	}

	if m := reGuildName.FindStringSubmatch(content); len(m) > 1 {
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

	if m := reWhatToVisit.FindStringSubmatch(content); len(m) > 1 {
		whatToVisit = cleanSection(m[1])
	}

	return
}

// ExtractName strips decorators and suffixes from a Discord thread name.
// e.g. "🏯 Iron Keep - Season 2" → "Iron Keep"
func ExtractName(threadName string) string {
	parts := strings.SplitN(threadName, " -", 2)
	return strings.TrimSpace(strings.Trim(parts[0], "[]🏯📍"))
}

// IsImage reports whether filename has a recognised image extension.
func IsImage(filename string) bool {
	switch strings.ToLower(fileExt(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// IsVideo reports whether filename has a recognised video extension.
func IsVideo(filename string) bool {
	switch strings.ToLower(fileExt(filename)) {
	case ".mp4", ".mov", ".webm", ".mkv", ".avi":
		return true
	}
	return false
}

func cleanSection(s string) string {
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

func fileExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
