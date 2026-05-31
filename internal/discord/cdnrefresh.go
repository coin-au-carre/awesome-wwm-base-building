package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// BulkRefreshURLs calls the Discord refresh-urls API to extend the expiry of
// the given cdn.discordapp.com attachment URLs. Returns original → refreshed.
func BulkRefreshURLs(token string, urls []string) (map[string]string, error) {
	body, err := json.Marshal(map[string]any{"attachment_urls": urls})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "https://discord.com/api/v10/attachments/refresh-urls", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord API %d: %s", resp.StatusCode, b)
	}
	var result struct {
		RefreshedURLs []struct {
			Original  string `json:"original"`
			Refreshed string `json:"refreshed"`
		} `json:"refreshed_urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	m := make(map[string]string, len(result.RefreshedURLs))
	for _, r := range result.RefreshedURLs {
		m[r.Original] = r.Refreshed
	}
	return m, nil
}

// ToCDNForm normalizes a Discord attachment URL to cdn.discordapp.com with
// only the auth params (ex/is/hm), as required by the refresh-urls API.
func ToCDNForm(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Host = "cdn.discordapp.com"
	q := u.Query()
	newQ := make(url.Values)
	for _, key := range []string{"ex", "is", "hm"} {
		if v := q.Get(key); v != "" {
			newQ.Set(key, v)
		}
	}
	u.RawQuery = newQ.Encode()
	return u.String(), nil
}
