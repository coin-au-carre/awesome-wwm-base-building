package blueprint

import (
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
		p.IsFree = true // no price field → free by default
	}

	if m := reMaterials.FindStringSubmatch(content); len(m) > 1 {
		p.Materials = strings.TrimSpace(m[1])
	}

	// Description: everything left after stripping structured field lines
	desc := reStripFields.ReplaceAllString(content, "")
	p.Description = strings.TrimSpace(desc)

	return p
}
