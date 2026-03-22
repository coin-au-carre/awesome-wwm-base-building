package main

import (
	"regexp"
	"strings"
)

var (
	reBracketID = regexp.MustCompile(`\[(\d+)\]`)
	reBuilders  = regexp.MustCompile(`(?i)builders?:\s*([^\n]*)`)
)

func parseFirstPost(content string) (id string, builders []string) {
	if m := reBracketID.FindStringSubmatch(content); len(m) > 1 {
		id = m[1]
	}
	if m := reBuilders.FindStringSubmatch(content); len(m) > 1 {
		for _, b := range strings.Split(m[1], ",") {
			b = strings.TrimSpace(b)
			if b != "" && !strings.HasPrefix(b, "#") && !strings.HasPrefix(b, ":") {
				builders = append(builders, b)
			}
		}
	}
	return
}

func extractGuildName(threadName string) string {
	parts := strings.SplitN(threadName, " -", 2)
	return strings.TrimSpace(strings.Trim(parts[0], "[]🏯📍"))
}

func isImage(filename string) bool {
	switch strings.ToLower(getExt(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

func getExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
