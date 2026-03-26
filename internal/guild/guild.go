package guild

type Guild struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	Builders         []string `json:"builders"`
	Tags             []string `json:"tags,omitempty"`
	DiscordThread    string   `json:"discordThread"`
	BuilderDiscordID string   `json:"builderDiscordId,omitempty"`
	Lore             string   `json:"lore,omitempty"`
	WhatToVisit      string   `json:"whatToVisit,omitempty"`
	Score            int      `json:"score"`
	Screenshots      []string `json:"screenshots,omitempty"`
}
