package interior

// Interior holds data for a single interior design thread.
type Interior struct {
	Name          string   `json:"name"`
	BuilderName   string   `json:"builderName,omitempty"`
	BuilderID     string   `json:"builderId,omitempty"`
	Description   string   `json:"description,omitempty"`
	Screenshots   []string `json:"screenshots,omitempty"`
	Videos        []string `json:"videos,omitempty"`
	DiscordThread string   `json:"discordThread"`
	CreatedAt     string   `json:"createdAt,omitempty"`
	LastModified  string   `json:"lastModified,omitempty"`

	LastScreenshotNotifiedAt string `json:"lastScreenshotNotifiedAt,omitempty"`
	LastVideoNotifiedAt      string `json:"lastVideoNotifiedAt,omitempty"`
}
