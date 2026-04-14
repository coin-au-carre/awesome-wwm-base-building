package guild

type ScreenshotSection struct {
	Label       string   `json:"label,omitempty"`
	Screenshots []string `json:"screenshots"`
}

type Guild struct {
	ID                 string              `json:"id,omitempty"`
	Name               string              `json:"name"`
	GuildName          string              `json:"guildName,omitempty"`
	Builders           []string            `json:"builders"`
	Tags               []string            `json:"tags,omitempty"`
	DiscordThread      string              `json:"discordThread"`
	BuilderDiscordID   string              `json:"builderDiscordId,omitempty"`
	Lore               string              `json:"lore,omitempty"`
	WhatToVisit        string              `json:"whatToVisit,omitempty"`
	Score              int                 `json:"score"`
	Note               string              `json:"note,omitempty"`
	CoverImage         string              `json:"coverImage,omitempty"`
	Screenshots        []string            `json:"screenshots,omitempty"`
	ScreenshotSections []ScreenshotSection `json:"screenshotSections,omitempty"`
	Videos             []string            `json:"videos,omitempty"`
	LastModified        string              `json:"lastModified,omitempty"`
	AllowedContributors []string            `json:"allowedContributors,omitempty"`
}
