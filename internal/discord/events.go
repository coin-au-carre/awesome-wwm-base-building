package discord

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
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
	var result parsedDesc
	var bodyLines []string
	headerDone := false

	for _, line := range lines {
		if headerDone {
			bodyLines = append(bodyLines, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			// Blank line ends the header block.
			headerDone = true
			continue
		}

		key, val, found := strings.Cut(trimmed, ":")
		if !found {
			// Not a key: value line — treat as body.
			headerDone = true
			bodyLines = append(bodyLines, line)
			continue
		}

		key = strings.TrimSpace(strings.ToLower(key))
		val = strings.TrimSpace(val)

		switch key {
		case "guild":
			result.guildName = val
		case "guild id":
			result.guildID = val
		case "type":
			if et, ok := validEventTypes[strings.ToLower(val)]; ok {
				result.eventType = et
			}
		default:
			// Unknown key — stop parsing header.
			headerDone = true
			bodyLines = append(bodyLines, line)
		}
	}

	result.description = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return result
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
