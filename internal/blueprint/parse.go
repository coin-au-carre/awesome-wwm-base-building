package blueprint

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// "BuilderName [ID]" or "Builder: BuilderName [ID]"
	reBuilder       = regexp.MustCompile(`(?im)^(?:builder\s*:[ \t]*)?([\p{L}\p{N}_.\- ]+?)\s*\[(\w+)\]\s*$`)
	reBuilderSimple = regexp.MustCompile(`(?im)^builder\s*:[ \t]*([^\n\[]+?)\s*(?:\[(\w+)\])?\s*$`)
	rePrice           = regexp.MustCompile(`(?im)^price\s*:[ \t]*([^\n]+)`)
	rePriceTag        = regexp.MustCompile(`(?i)!price\s+([^!\n]+)`)
	rePriceEchoPearls = regexp.MustCompile(`(?im)^\s*(\d[\d\s,]*\s*echo\s+pearls?)\s*$`)
	reMaterials       = regexp.MustCompile(`(?im)^materials?\s*:[ \t]*([^\n]+)`)
	reCover           = regexp.MustCompile(`(?i)cover:[ \t]*(\d+)`)
	reShareCode       = regexp.MustCompile(`(?i)\bSHARE[0-9a-f]{16}\b`)
	// matches payment mentions in prose: "60 pearls", "180 echo beads", "$5", "30 dollars"
	RePayInProse = regexp.MustCompile(`(?i)\d+\s*(?:echo\s+)?(?:pearl|bead|dollar)s?|\$\s*\d`)

	// strips structured field lines to isolate the freeform description body
	reStripFields = regexp.MustCompile(`(?im)^(?:builder\s*:[ \t]*[^\n]*|price\s*:[ \t]*[^\n]*|materials?\s*:[ \t]*[^\n]*|[\p{L}\p{N}_.\- ]+\s*\[\w+\])\n?`)
)

// ParsedBlueprintPost holds structured data from the first post of a blueprint thread.
type ParsedBlueprintPost struct {
	BuilderName  string
	BuilderID    string
	Price        string
	IsFree       bool
	IsPayToBuild bool
	Materials    string
	Description  string
	ShareCodes   []string
	CoverIdx     int // 1-based; 0 means not specified
}

// ParseFirstPost extracts optional structured fields from a blueprint thread's first post.
func ParseFirstPost(content string) ParsedBlueprintPost {
	var p ParsedBlueprintPost

	// Try "Builder: Name [ID]" format first, then bare "Name [ID]"
	if m := reBuilderSimple.FindStringSubmatch(content); len(m) > 1 {
		p.BuilderName = strings.TrimSpace(m[1])
		if len(m) > 2 {
			p.BuilderID = strings.TrimSpace(m[2])
		}
	} else if m := reBuilder.FindStringSubmatch(content); len(m) > 1 {
		p.BuilderName = strings.TrimSpace(m[1])
		if len(m) > 2 {
			p.BuilderID = strings.TrimSpace(m[2])
		}
	}

	if m := rePrice.FindStringSubmatch(content); len(m) > 1 {
		raw := strings.TrimSpace(m[1])
		p.Price = raw
		lower := strings.ToLower(raw)
		p.IsFree = strings.Contains(lower, "free")
		p.IsPayToBuild = lower != "free" && lower != ""
	} else if m := rePriceTag.FindStringSubmatch(content); len(m) > 1 {
		raw := strings.TrimSpace(m[1])
		p.Price = raw
		lower := strings.ToLower(raw)
		p.IsFree = strings.Contains(lower, "free")
		p.IsPayToBuild = lower != "free" && lower != ""
	} else if m := rePriceEchoPearls.FindStringSubmatch(content); len(m) > 1 {
		raw := strings.TrimSpace(m[1])
		p.Price = raw
		p.IsPayToBuild = true
	} else {
		lower := strings.ToLower(content)
		if RePayInProse.MatchString(content) {
			p.IsPayToBuild = true
		} else if strings.Contains(lower, "free") {
			p.IsFree = true
		} else {
			p.IsPayToBuild = true // no "free" and no structured price → paid by default
		}
	}

	if m := reMaterials.FindStringSubmatch(content); len(m) > 1 {
		p.Materials = strings.TrimSpace(m[1])
	}

	if m := reCover.FindStringSubmatch(content); len(m) > 1 {
		fmt.Sscan(m[1], &p.CoverIdx)
	}

	if codes := reShareCode.FindAllString(content, -1); len(codes) > 0 {
		seen := make(map[string]bool, len(codes))
		for _, c := range codes {
			upper := strings.ToUpper(c)
			if !seen[upper] {
				seen[upper] = true
				p.ShareCodes = append(p.ShareCodes, upper)
			}
		}
	}

	// Description: everything left after stripping structured field lines
	desc := reStripFields.ReplaceAllString(content, "")
	p.Description = strings.TrimSpace(desc)

	return p
}
