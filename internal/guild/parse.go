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
	reBuilders           = regexp.MustCompile(`(?i)builders?:[ \t]*([^\n]*)`)
	reBuildersBulkBlock  = regexp.MustCompile(`(?i)builders?:[ \t]*\n((?:[ \t]*-[^\n]+\n?)*)`)
	reAdditionalCredits  = regexp.MustCompile(`(?i)^additional\s+credits?\s+to\s+(.+)`)
	reGuildName       = regexp.MustCompile(`(?m)^[#\s]*(?::[^:]+:|\*\*|\p{So}[\x{FE0E}\x{FE0F}]?\s*|\x{FE0F}\s*)*(?:[A-Za-z]+:\s*)?(.+?)\**\s*[\[(]\d{6,9}[\])]`)
	reGuildNameEq     = regexp.MustCompile(`🏯[^=\n]*=\s*([^\n]+)`)
	reLore            = regexp.MustCompile(`(?im)(?:^###[^\n]*lore|\*\*\s*lore\s*\*\*|^[^\S\n]*(?:\p{So}[\x{FE0E}\x{FE0F}]?[^\S\n]*)?\blore\b)[^\n]*\n+([\s\S]*?)(?:\p{So}[\x{FE0E}\x{FE0F}]?\s*)?(?:\*\*\s*(?:what|places)\s+to\s+visit\s*\*\*|\b(?:what|places)\s+to\s+visit\b|⚠️|^###|\z)`)
	reLoreEq          = regexp.MustCompile(`(?im)\blore\b\s*=\s*([^\n]+)`)
	reLoreInline      = regexp.MustCompile("(?im)^[^\\S\\n]*(?:\\p{So}[\\x{FE0E}\\x{FE0F}]?[^\\S\\n]*)?\\blore\\b[^\\S\\n]*[`=:]?[^\\S\\n]*(\\S[^\\n]*)")
	reWhatToVisit     = regexp.MustCompile(`(?im)(?:^###[^\n]*(?:what|places)\s+to\s+visit|\*\*\s*(?:what|places)\s+to\s+visit\s*\*\*|\b(?:what|places)\s+to\s+visit\b)[^\n]*\n+([\s\S]*?)(?:🗳|⚠️|^###|\z)`)
	reWhatToVisitEq   = regexp.MustCompile(`(?im)what\s+to\s+visit\s*=\s*([^\n]+)`)
	reWhatToVisitInline = regexp.MustCompile("(?im)^[^\\S\\n]*(?:\\p{So}[^\\S\\n]*)?\\b(?:what|places)\\s+to\\s+visit\\b[^\\S\\n]*[`=:]?[^\\S\\n]*(\\S[^\\n]*)")
	reCover           = regexp.MustCompile(`(?i)cover:[ \t]*(\d+)`)
	reCoverStrip      = regexp.MustCompile(`(?im)[\n\r]*[ \t]*cover:[ \t]*\d+[ \t]*$`)
	reTrailingStars   = regexp.MustCompile(`(?:\s*\n\s*\*+)+\s*$`)
	reBuildTitleColon   = regexp.MustCompile(`\s*:\s*`) // "GuildName: Build Title" and spacing variants
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
		if inline := strings.TrimSpace(m[1]); inline != "" {
			for b := range strings.SplitSeq(inline, ",") {
				b = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(b), "*"))
				if b != "" && !strings.HasPrefix(b, "#") && !strings.HasPrefix(b, ":") {
					p.Builders = append(p.Builders, b)
				}
			}
		} else if m2 := reBuildersBulkBlock.FindStringSubmatch(content); len(m2) > 1 {
			for line := range strings.SplitSeq(m2[1], "\n") {
				p.Builders = append(p.Builders, parseBuilderBullet(line)...)
			}
		}
	}

	if m := reLore.FindStringSubmatch(content); len(m) > 1 {
		p.Lore = CleanSection(m[1])
	}
	if p.Lore == "" {
		if m := reLoreInline.FindStringSubmatch(content); len(m) > 1 {
			p.Lore = CleanSection(m[1])
		}
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
		if m := reWhatToVisitInline.FindStringSubmatch(content); len(m) > 1 {
			p.WhatToVisit = CleanSection(m[1])
		}
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

	// Fallback for solo posts: no structured sections, long free-form lore.
	if p.Lore == "" && len(p.Builders) == 0 && p.WhatToVisit == "" && len(strings.TrimSpace(content)) > 300 {
		p.Lore = CleanSection(content)
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
// and an optional build title from a separator suffix.
// Preferred separator: "GuildName: Build Title" (colon, any spacing around it).
// Legacy separator:    "GuildName - Build Title" (space-dash).
// e.g. "WITCHERS [10248427" → ("WITCHERS", "10248427", "")
// e.g. "🏯 Mutiny: Lost World" → ("Mutiny", "", "Lost World")
// e.g. "🏯 Mutiny - Lost World" → ("Mutiny", "", "Lost World")
func ExtractNameAndID(threadName string) (name, id, buildTitle string) {
	var raw string
	if loc := reBuildTitleColon.FindStringIndex(threadName); loc != nil {
		raw = strings.TrimSpace(threadName[:loc[0]])
		buildTitle = strings.TrimSpace(threadName[loc[1]:])
	} else if parts := strings.SplitN(threadName, " -", 2); len(parts) == 2 {
		raw = strings.TrimSpace(parts[0])
		buildTitle = strings.TrimSpace(parts[1])
	} else {
		raw = strings.TrimSpace(threadName)
	}
	raw = strings.TrimSpace(strings.TrimLeft(raw, "#🏯📍️"))
	if m := reThreadID.FindStringSubmatch(raw); len(m) == 2 {
		id = m[1]
		raw = reThreadID.ReplaceAllString(raw, "")
	} else if m := reThreadID.FindStringSubmatch(buildTitle); len(m) == 2 {
		id = m[1]
		buildTitle = strings.TrimSpace(reThreadID.ReplaceAllString(buildTitle, ""))
	}
	name = strings.TrimSpace(strings.Trim(raw, " 🏯📍️"))
	// Only strip a leading "[" / trailing "]" when they wrap the whole name as a
	// matched pair (e.g. a stray ID bracket leftover); a single bracket on just
	// one side is part of the name's actual content (e.g. "[Tag] rest of name").
	if strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]") {
		name = strings.TrimSpace(name[1 : len(name)-1])
	}
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

func parseBuilderBullet(line string) []string {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
	if line == "" {
		return nil
	}
	if m := reAdditionalCredits.FindStringSubmatch(line); len(m) > 1 {
		raw := strings.ReplaceAll(m[1], " and ", ",")
		var names []string
		for n := range strings.SplitSeq(raw, ",") {
			if n = strings.TrimSpace(n); n != "" {
				names = append(names, n)
			}
		}
		return names
	}
	if i := strings.IndexAny(line, "[|"); i >= 0 {
		line = line[:i]
	}
	if name := strings.TrimSpace(line); name != "" {
		return []string{name}
	}
	return nil
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

