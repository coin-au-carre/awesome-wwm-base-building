package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BulkRefreshURLs calls the Discord refresh-urls API to extend the expiry of
// the given cdn.discordapp.com attachment URLs. Returns original → refreshed.
// Batches automatically at 50 URLs (the API maximum).
func BulkRefreshURLs(token string, urls []string) (map[string]string, error) {
	m := make(map[string]string, len(urls))
	for i := 0; i < len(urls); i += 50 {
		batch := urls[i:min(i+50, len(urls))]
		bm, err := refreshBatch(token, batch)
		if err != nil {
			return nil, err
		}
		for k, v := range bm {
			m[k] = v
		}
	}
	return m, nil
}

func refreshBatch(token string, urls []string) (map[string]string, error) {
	body, err := json.Marshal(map[string]any{"attachment_urls": urls})
	if err != nil {
		return nil, err
	}
	// This hits the Discord API directly over plain net/http (not through
	// discordgo's session, which has its own rate limiter), so 429s need
	// handling here. retry_after is well under a second in practice.
	const maxAttempts = 5
	for attempt := 1; ; attempt++ {
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
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts {
			var rl struct {
				RetryAfter float64 `json:"retry_after"`
			}
			json.NewDecoder(resp.Body).Decode(&rl)
			resp.Body.Close()
			wait := time.Duration(rl.RetryAfter * float64(time.Second))
			if wait <= 0 {
				wait = time.Second
			}
			time.Sleep(wait)
			continue
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
