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
	reLore            = regexp.MustCompile(`(?im)(?:^###[^\n]*lore|\*\*\s*lore\s*\*\*|\blore\b)[^\n]*\n+([\s\S]*?)(?:\p{So}\s*)?(?:\*\*\s*what\s+to\s+visit\s*\*\*|\bwhat\s+to\s+visit\b|⚠️|^###|\z)`)
	reLoreEq          = regexp.MustCompile(`(?im)\blore\b\s*=\s*([^\n]+)`)
	reWhatToVisit     = regexp.MustCompile(`(?im)(?:^###[^\n]*what\s+to\s+visit|\*\*\s*what\s+to\s+visit\s*\*\*|\bwhat\s+to\s+visit\b)[^\n]*\n+([\s\S]*?)(?:🗳|⚠️|^###|\z)`)
	reWhatToVisitEq   = regexp.MustCompile(`(?im)what\s+to\s+visit\s*=\s*([^\n]+)`)
	reCover           = regexp.MustCompile(`(?i)cover:[ \t]*(\d+)`)
	reCoverStrip      = regexp.MustCompile(`(?im)[\n\r]*[ \t]*cover:[ \t]*\d+[ \t]*$`)
	reTrailingStars   = regexp.MustCompile(`(?:\s*\n\s*\*+)+\s*$`)
	reOnBehalf          = regexp.MustCompile(`(?i)on behalf of\s+@([\w.]+)`)
	reOnBehalfSnowflake = regexp.MustCompile(`(?i)on behalf of\s+<@(\d+)>`)
	reOnBehalfPresent   = regexp.MustCompile(`(?i)on behalf`)
	reBuildTitle        = regexp.MustCompile(`(?i)build\s+title\s*:[ \t]*([^\n]+)`)
	reIsCurrent         = regexp.MustCompile(`(?i)current\s*:[ \t]*(yes|true|1)\b`)
	reHostedAt          = regexp.MustCompile(`(?im)^hosted\s+at\s*:[ \t]*([^\n\[]+?)(?:[ \t]*[\[(](\d+)[\])])?[ \t]*$`)
)

var skipPhrases = []string{
	"replace_with_your_lore",
	"describe_point_of_interest",
	"claim ownership",
	"contact us",
}

// ParsedPost holds all structured data extracted from a Discord thread's first post.
type ParsedPost struct {
	ID               string
	GuildName        string
	Builders         []string
	Lore             string
	WhatToVisit      string
	CoverIdx         int // 1-based; 0 means not specified
	PostedOnBehalfOf string // empty when not a behalf post; "unknown" when "on behalf" is present but username cannot be parsed
	BuildTitle       string
	IsCurrent        bool
	HostedAtGuildName string
	HostedAtGuildID   string
}

// ParseFirstPost extracts structured data from the first message of a Discord thread.
func ParseFirstPost(content string) ParsedPost {
	var p ParsedPost

	if m := reBracketID.FindStringSubmatch(content); len(m) > 1 {
		p.ID = m[1]
	} else if m := reEightDigit.FindStringSubmatch(content); len(m) > 1 {
		p.ID = m[1]
	}

	if m := reGuildName.FindStringSubmatch(content); len(m) > 1 {
		p.GuildName = strings.TrimSpace(m[1])
	} else if m := reGuildNameEq.FindStringSubmatch(content); len(m) > 1 {
		p.GuildName = strings.TrimSpace(m[1])
	}

	if m := reBuilders.FindStringSubmatch(content); len(m) > 1 {
		for _, b := range strings.Split(m[1], ",") {
			b = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(b), "*"))
			if b != "" && !strings.HasPrefix(b, "#") && !strings.HasPrefix(b, ":") {
				p.Builders = append(p.Builders, b)
			}
		}
	}

	if m := reLore.FindStringSubmatch(content); len(m) > 1 {
		p.Lore = CleanSection(m[1])
	}
	if p.Lore == "" {
		if m := reLoreEq.FindStringSubmatch(content); len(m) > 1 {
			p.Lore = CleanSection(m[1])
		}
	}

	if m := reWhatToVisit.FindStringSubmatch(content); len(m) > 1 {
		p.WhatToVisit = CleanSection(m[1])
	}
	if p.WhatToVisit == "" {
		if m := reWhatToVisitEq.FindStringSubmatch(content); len(m) > 1 {
			p.WhatToVisit = CleanSection(m[1])
		}
	}

	if m := reCover.FindStringSubmatch(content); len(m) > 1 {
		fmt.Sscan(m[1], &p.CoverIdx)
	}

	if m := reOnBehalf.FindStringSubmatch(content); len(m) > 1 {
		p.PostedOnBehalfOf = m[1]
	} else if m := reOnBehalfSnowflake.FindStringSubmatch(content); len(m) > 1 {
		p.PostedOnBehalfOf = m[1]
	} else if reOnBehalfPresent.MatchString(content) {
		p.PostedOnBehalfOf = "unknown"
	}

	if m := reBuildTitle.FindStringSubmatch(content); len(m) > 1 {
		p.BuildTitle = strings.TrimSpace(m[1])
	}

	p.IsCurrent = reIsCurrent.MatchString(content)

	if m := reHostedAt.FindStringSubmatch(content); len(m) > 1 {
		p.HostedAtGuildName = strings.TrimSpace(m[1])
		if len(m) > 2 {
			p.HostedAtGuildID = strings.TrimSpace(m[2])
		}
	}

	return p
}

// ExtractNameAndID strips decorators/suffixes from a Discord thread name and
// extracts an optional numeric guild ID embedded as a trailing bracket token,
// and an optional build title from the " - Subtitle" suffix.
// e.g. "WITCHERS [10248427" → ("WITCHERS", "10248427", "")
// e.g. "🏯 Mutiny - Lost World" → ("Mutiny", "", "Lost World")
func ExtractNameAndID(threadName string) (name, id, buildTitle string) {
	parts := strings.SplitN(threadName, " -", 2)
	raw := strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		buildTitle = strings.TrimSpace(parts[1])
	}
	raw = strings.TrimSpace(strings.TrimLeft(raw, "#🏯📍"))
	if m := reThreadID.FindStringSubmatch(raw); len(m) == 2 {
		id = m[1]
		raw = reThreadID.ReplaceAllString(raw, "")
	}
	name = strings.TrimSpace(strings.Trim(raw, "[] 🏯📍"))
	name = strings.TrimSpace(reEightDigitName.ReplaceAllString(name, ""))
	return
}

// ExtractName strips decorators and suffixes from a Discord thread name.
// e.g. "🏯 Iron Keep - Season 2" → "Iron Keep"
func ExtractName(threadName string) string {
	name, _, _ := ExtractNameAndID(threadName)
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

func CleanSection(s string) string {
	s = strings.TrimSpace(s)
	s = reCoverStrip.ReplaceAllString(s, "")
	s = reTrailingStars.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if len(s) < 3 {
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

