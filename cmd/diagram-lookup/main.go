// cmd/diagram-lookup/main.go — look up a diagram's title/description/
// screenshot/creation date from its ART code. No credentials needed;
// the backend (host configured via DIAGRAM_LOOKUP_API_BASE) is a fully
// public, unauthenticated endpoint.
//
// SHARE codes cannot be resolved this way — paste them directly into the
// game client instead.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ruby/internal/cmdutil"
)

type planInfo struct {
	Name       string  `json:"name"`
	Msg        string  `json:"msg"`
	PictureURL string  `json:"picture_url"`
	UploadTs   float64 `json:"upload_ts"`
	HeatVal    int     `json:"heat_val"`
	Tags       []int   `json:"tags"`
	Pid        string  `json:"pid"`
}

type designerInfo struct {
	FollowerNum int      `json:"follower_num"`
	LikeNum     int      `json:"like_num"`
	PlansPublic []string `json:"plans_public"`
}

// postJSON POSTs body to apiBase+path, optionally with region headers
// (CN and Global codes/designers live on separate backend infrastructure
// and don't fall back to each other server-side), and decodes the JSON
// response into out.
func postJSON(apiBase, path, region string, body any, out any) error {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, apiBase+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if region != "" {
		req.Header.Set("X-Target-Server", region)
		req.Header.Set("X-Server", region)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchPlan(apiBase, code, region string) (planInfo, bool, error) {
	var parsed struct {
		Result map[string]planInfo `json:"result"`
	}
	if err := postJSON(apiBase, "/get_face_plan_brief_info_batch", region,
		map[string]any{"uid": "", "plan_id_dict": map[string]int{code: 0}}, &parsed); err != nil {
		return planInfo{}, false, err
	}
	info, ok := parsed.Result[code]
	return info, ok, nil
}

func fetchDesigner(apiBase, pid, region string) (designerInfo, bool, error) {
	var parsed struct {
		Result map[string]designerInfo `json:"result"`
	}
	if err := postJSON(apiBase, "/get_face_designer_brief_info_batch", region,
		map[string]any{"uid": "", "pid_dict": map[string]int{pid: 0}}, &parsed); err != nil {
		return designerInfo{}, false, err
	}
	info, ok := parsed.Result[pid]
	return info, ok, nil
}

// fetchNickname returns the designer's real display name. The endpoint's
// "fields" parameter must be exactly ["base","head"] (not e.g.
// "nickname" directly) — the name comes back nested inside "base".
func fetchNickname(apiBase, pid, region string) (string, bool) {
	var parsed struct {
		Result map[string]struct {
			Base struct {
				Nickname string `json:"nickname"`
			} `json:"base"`
		} `json:"result"`
	}
	err := postJSON(apiBase, "/get_players_info", region,
		map[string]any{"fields": []string{"base", "head"}, "hostnum2pids": map[string][]string{"0": {pid}}}, &parsed)
	if err != nil {
		return "", false
	}
	name := parsed.Result[pid].Base.Nickname
	return name, name != ""
}

func main() {
	root := flag.String("root", cmdutil.RootDir(), "root directory containing .env")
	flag.Parse()

	cmdutil.LoadEnv(*root)

	code := flag.Arg(0)
	if code == "" {
		fmt.Fprintln(os.Stderr, "usage: diagram-lookup <ART code>")
		os.Exit(2)
	}
	if strings.HasPrefix(strings.ToUpper(code), "SHARE") {
		fmt.Fprintln(os.Stderr, "SHARE codes can't be looked up externally — no known endpoint resolves them.")
		fmt.Fprintln(os.Stderr, "Paste it directly into the game client instead.")
		os.Exit(1)
	}
	code = strings.TrimPrefix(code, "ART")

	apiBase := cmdutil.RequireEnv("DIAGRAM_LOOKUP_API_BASE")

	// The two regions live on entirely separate backend infrastructure;
	// a code only ever resolves against its own region, so try Global
	// first, then fall back to the default (CN-hosted) lookup.
	region := "global"
	info, ok, err := fetchPlan(apiBase, code, region)
	if err == nil && !ok {
		region = ""
		info, ok, err = fetchPlan(apiBase, code, region)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "lookup failed: %v\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "no data found for %q — the code may be expired, private, mistyped, or (for a very recently created/imported diagram) not yet catalogued.\n", code)
		os.Exit(1)
	}

	fmt.Printf("Title:       %s\n", info.Name)
	fmt.Printf("Description: %s\n", info.Msg)
	fmt.Printf("Screenshot:  %s\n", info.PictureURL)
	if info.UploadTs > 0 {
		fmt.Printf("Uploaded:    %s\n", time.Unix(int64(info.UploadTs), 0).UTC().Format(time.RFC3339))
	}
	fmt.Printf("Heat:        %d\n", info.HeatVal)
	fmt.Printf("Tags:        %v\n", info.Tags)

	if info.Pid == "" {
		return
	}
	who := info.Pid
	if name, ok := fetchNickname(apiBase, info.Pid, region); ok {
		who = name
	}
	published, followers, likes := "?", "?", "?"
	if designer, ok, err := fetchDesigner(apiBase, info.Pid, region); err == nil && ok {
		published = fmt.Sprint(len(designer.PlansPublic))
		followers = fmt.Sprint(designer.FollowerNum)
		likes = fmt.Sprint(designer.LikeNum)
	}
	fmt.Printf("Designer:    %s  (%s published, %s followers, %s likes)\n", who, published, followers, likes)
}
