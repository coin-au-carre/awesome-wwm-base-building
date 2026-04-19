package discord

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	reLocBracket = regexp.MustCompile(`^(.*?)\s*[\[(](?:ID\s+)?(\d+)[\])]?\s*$`)
	reLocSpaceID = regexp.MustCompile(`^(.*?)\s+(\d+)\s*$`)
)

// EventStatus mirrors Discord scheduled event status values.
type EventStatus string

const (
	EventStatusScheduled EventStatus = "scheduled"
	EventStatusActive    EventStatus = "active"
	EventStatusCompleted EventStatus = "completed"
	EventStatusCanceled  EventStatus = "canceled"
)

// EventType classifies the kind of activity being organised.
type EventType string

const (
	EventTypeTour     EventType = "tour"
	EventTypePVP      EventType = "pvp"
	EventTypeMarriage EventType = "marriage"
	EventTypeDancing  EventType = "dancing"
	EventTypeFashion  EventType = "fashion"
	EventTypeContest  EventType = "contest"
	EventTypeRace     EventType = "race"
	EventTypeOther    EventType = "other"
)

var validEventTypes = map[string]EventType{
	"tour":     EventTypeTour,
	"pvp":      EventTypePVP,
	"marriage": EventTypeMarriage,
	"dancing":  EventTypeDancing,
	"fashion":  EventTypeFashion,
	"contest":  EventTypeContest,
	"race":     EventTypeRace,
	"other":    EventTypeOther,
}

// Event is the serialisable form written to data/events.json.
type Event struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description,omitempty"`
	GuildName       string      `json:"guildName,omitempty"`
	GuildID         string      `json:"guildId,omitempty"`
	Type            EventType   `json:"type,omitempty"`
	ScheduledStart  time.Time   `json:"scheduledStart"`
	ScheduledEnd    *time.Time  `json:"scheduledEnd,omitempty"`
	Location        string      `json:"location,omitempty"`
	Status          EventStatus `json:"status"`
	SubscriberCount int         `json:"subscriberCount"`
	DiscordURL      string      `json:"discordUrl"`
	Image           string      `json:"image,omitempty"`
}

// parsedDesc holds the result of parsing a structured event description.
type parsedDesc struct {
	guildName   string
	guildID     string
	eventType   EventType
	description string
}

// parseDescription extracts structured header fields from the Discord event description.
//
// Expected format (all fields optional):
//
//	Guild: Iron Fortress
//	Guild ID: iron-fortress
//	Type: pvp
//
//	Free text description shown on the website.
//
// Lines that start with a known key are consumed as metadata; remaining lines
// (after stripping leading blank lines) become the display description.
func parseDescription(raw string) parsedDesc {
	lines := strings.Split(raw, "\n")

	// Try top header first.
	if pd, body, ok := extractMetaBlock(lines, true); ok {
		pd.description = strings.TrimSpace(strings.Join(body, "\n"))
		return pd
	}
	// Fall back to bottom footer.
	if pd, body, ok := extractMetaBlock(lines, false); ok {
		pd.description = strings.TrimSpace(strings.Join(body, "\n"))
		return pd
	}
	return parsedDesc{description: strings.TrimSpace(raw)}
}

// extractMetaBlock scans for a contiguous block of "Key: value" metadata lines
// at the top (fromTop=true) or bottom (fromTop=false) of lines. A blank line
// terminates the block. Returns the parsed fields, the remaining body lines,
// and true when at least one recognised field was found.
func extractMetaBlock(lines []string, fromTop bool) (parsedDesc, []string, bool) {
	type indexedLine struct {
		idx  int
		text string
	}

	// Build an ordered slice of (index, trimmed) for the candidate block.
	var candidate []indexedLine
	if fromTop {
		for i, l := range lines {
			t := strings.TrimSpace(l)
			if t == "" {
				break
			}
			candidate = append(candidate, indexedLine{i, t})
		}
	} else {
		// Walk from the bottom, skip trailing blank lines first.
		end := len(lines)
		for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
			end--
		}
		for i := end - 1; i >= 0; i-- {
			t := strings.TrimSpace(lines[i])
			if t == "" {
				break
			}
			candidate = append([]indexedLine{{i, t}}, candidate...)
		}
	}

	var pd parsedDesc
	blockIndices := map[int]bool{}
	for _, cl := range candidate {
		key, val, found := strings.Cut(cl.text, ":")
		if !found {
			break
		}
		key = strings.TrimSpace(strings.ToLower(key))
		val = strings.TrimSpace(val)
		switch key {
		case "guild":
			pd.guildName = val
			blockIndices[cl.idx] = true
		case "guild id":
			pd.guildID = val
			blockIndices[cl.idx] = true
		case "type":
			if et, ok := validEventTypes[strings.ToLower(val)]; ok {
				pd.eventType = et
			}
			blockIndices[cl.idx] = true
		default:
			// Unknown key — stop consuming.
		}
	}

	if len(blockIndices) == 0 {
		return pd, lines, false
	}

	var body []string
	for i, l := range lines {
		if !blockIndices[i] {
			body = append(body, l)
		}
	}
	// Trim the blank line that separated block from body.
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}
	for len(body) > 0 && strings.TrimSpace(body[0]) == "" {
		body = body[1:]
	}
	return pd, body, true
}

// parseLocation tries to extract a guild name and optional numeric ID from the
// event location string. Supported formats:
//
//	"Guild Name [12345678]" or "Guild Name (12345678)" — bracket/paren
//	"Guild Name (ID 12345678)"                         — paren with ID prefix
//	"Guild Name 12345678"                              — space-separated
func parseLocation(loc string) (guildName, guildID string) {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return
	}
	if m := reLocBracket.FindStringSubmatch(loc); len(m) == 3 {
		return strings.TrimSpace(m[1]), m[2]
	}
	if m := reLocSpaceID.FindStringSubmatch(loc); len(m) == 3 {
		return strings.TrimSpace(m[1]), m[2]
	}
	return loc, ""
}

func discordStatus(s discordgo.GuildScheduledEventStatus) EventStatus {
	switch s {
	case discordgo.GuildScheduledEventStatusActive:
		return EventStatusActive
	case discordgo.GuildScheduledEventStatusCompleted:
		return EventStatusCompleted
	case discordgo.GuildScheduledEventStatusCanceled:
		return EventStatusCanceled
	default:
		return EventStatusScheduled
	}
}

// FetchEvents returns all scheduled events for the given guild.
func FetchEvents(s *discordgo.Session, guildID string) ([]Event, error) {
	raw, err := s.GuildScheduledEvents(guildID, true)
	if err != nil {
		return nil, fmt.Errorf("fetching scheduled events: %w", err)
	}

	events := make([]Event, 0, len(raw))
	for _, e := range raw {
		location := e.EntityMetadata.Location

		var scheduledEnd *time.Time
		if e.ScheduledEndTime != nil {
			t := *e.ScheduledEndTime
			scheduledEnd = &t
		}

		parsed := parseDescription(e.Description)

		if parsed.guildName == "" && location != "" {
			locName, locID := parseLocation(location)
			parsed.guildName = locName
			if parsed.guildID == "" {
				parsed.guildID = locID
			}
		}

		var imageURL string
		if e.Image != "" {
			imageURL = fmt.Sprintf("https://cdn.discordapp.com/guild-events/%s/%s.png?size=512", e.ID, e.Image)
		}

		events = append(events, Event{
			ID:              e.ID,
			Name:            e.Name,
			Description:     parsed.description,
			GuildName:       parsed.guildName,
			GuildID:         parsed.guildID,
			Type:            parsed.eventType,
			ScheduledStart:  e.ScheduledStartTime,
			ScheduledEnd:    scheduledEnd,
			Location:        location,
			Status:          discordStatus(e.Status),
			SubscriberCount: e.UserCount,
			DiscordURL:      fmt.Sprintf("https://discord.com/events/%s/%s", guildID, e.ID),
			Image:           imageURL,
		})
	}

	return events, nil
}
