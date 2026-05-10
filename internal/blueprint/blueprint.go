package blueprint

import "ruby/internal/guild"

// Blueprint holds structured data for a single blueprint/diagram thread.
type Blueprint struct {
	Name               string                    `json:"name"`
	BuilderName        string                    `json:"builderName,omitempty"`
	BuilderID          string                    `json:"builderId,omitempty"`
	Price              string                    `json:"price,omitempty"`
	IsFree             bool                      `json:"isFree,omitempty"`
	IsPayToBuild       bool                      `json:"isPayToBuild,omitempty"`
	Materials          string                    `json:"materials,omitempty"`
	Description        string                    `json:"description,omitempty"`
	Tags               []string                  `json:"tags,omitempty"`
	Score              int                       `json:"score"`
	CoverImage         string                    `json:"coverImage,omitempty"`
	Screenshots        []string                  `json:"screenshots,omitempty"`
	ScreenshotSections []guild.ScreenshotSection  `json:"screenshotSections,omitempty"`
	Videos             []string                  `json:"videos,omitempty"`
	DiscordThread      string                    `json:"discordThread"`
	CreatedAt          string                    `json:"createdAt,omitempty"`
	LastModified       string                    `json:"lastModified,omitempty"`
	// internal notification cooldown tracking
	LastScreenshotNotifiedAt string `json:"lastScreenshotNotifiedAt,omitempty"`
	LastVideoNotifiedAt      string `json:"lastVideoNotifiedAt,omitempty"`
}
