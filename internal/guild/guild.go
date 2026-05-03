package guild

import "encoding/json"

type ScreenshotSection struct {
	Label       string   `json:"label,omitempty"`
	Screenshots []string `json:"screenshots"`
}

// Note holds either a plain string or a structured deletion record.
type Note struct {
	Text            string
	Status          string `json:"status,omitempty"`
	DiscordThread   string `json:"discordThread,omitempty"`
	PosterDiscordID string `json:"posterDiscordId,omitempty"`
}

func (n Note) IsZero() bool {
	return n.Text == "" && n.Status == "" && n.DiscordThread == "" && n.PosterDiscordID == ""
}

func (n Note) MarshalJSON() ([]byte, error) {
	if n.Status != "" || n.DiscordThread != "" || n.PosterDiscordID != "" {
		return json.Marshal(struct {
			Status          string `json:"status,omitempty"`
			DiscordThread   string `json:"discordThread,omitempty"`
			PosterDiscordID string `json:"posterDiscordId,omitempty"`
		}{n.Status, n.DiscordThread, n.PosterDiscordID})
	}
	return json.Marshal(n.Text)
}

func (n *Note) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &n.Text)
	}
	type noteObj Note
	return json.Unmarshal(data, (*noteObj)(n))
}

type Guild struct {
	ID                       string              `json:"id,omitempty"`
	Name                     string              `json:"name"`
	GuildName                string              `json:"guildName,omitempty"`
	Builders                 []string            `json:"builders"`
	Tags                     []string            `json:"tags,omitempty"`
	DiscordThread            string              `json:"discordThread"`
	PosterDiscordID          string              `json:"posterDiscordId,omitempty"`
	PosterUsername           string              `json:"posterUsername,omitempty"`
	Lore                     string              `json:"lore,omitempty"`
	WhatToVisit              string              `json:"whatToVisit,omitempty"`
	Score                    int                 `json:"score"`
	Note                     *Note               `json:"note,omitempty"`
	CoverImage               string              `json:"coverImage,omitempty"`
	Screenshots              []string            `json:"screenshots,omitempty"`
	ScreenshotSections       []ScreenshotSection `json:"screenshotSections,omitempty"`
	Videos                   []string            `json:"videos,omitempty"`
	IgnoredVideos            []string            `json:"ignoredVideos,omitempty"`
	FormerNames              []string            `json:"formerNames,omitempty"`
	CreatedAt                string              `json:"createdAt,omitempty"`
	LastModified             string              `json:"lastModified,omitempty"`
	LastScreenshotNotifiedAt string              `json:"lastScreenshotNotifiedAt,omitempty"`
	LastVideoNotifiedAt      string              `json:"lastVideoNotifiedAt,omitempty"`
	AllowedContributors      []string            `json:"allowedContributors,omitempty"`
	PostedOnBehalfOf         string              `json:"postedOnBehalfOf,omitempty"`
	ScoutedByDiscordID       string              `json:"scoutedByDiscordId,omitempty"`
	BuildTitle               string              `json:"buildTitle,omitempty"`
	IsCurrent                bool                `json:"isCurrent,omitempty"`
	HostedAtGuildName        string              `json:"hostedAtGuildName,omitempty"`
	HostedAtGuildID          string              `json:"hostedAtGuildId,omitempty"`
}
