// Package netease calls NetEase's real, private WWM game-backend API
// directly — the same one wbm-relay talks to (see the wbm-relay repo's
// wbm-tool/gallery-api.md for the full protocol reference this was
// reverse-engineered from). Only the one read used by the builder-identity
// system lives here: resolving a public number_id to its internal
// pid/hostnum and current in-game nickname.
package netease

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/vmihailenco/msgpack/v5"
)

const findPeopleByNumberIDURL = "https://h72naxx2gb-ms-prod.easebar.com/flk/find_people/by_number_id"

// ErrNotFound means the number_id doesn't resolve to any real account.
var ErrNotFound = errors.New("netease: number_id not found")

// PlayerRef is a NetEase player resolved from their public number_id.
type PlayerRef struct {
	PID      string
	Hostnum  int
	Nickname string
}

// ResolveByNumberID resolves a NetEase account's public number_id (the id
// shown on a player's card in-game) to its internal pid/hostnum and current
// nickname. No auth needed — confirmed live 2026-07-20, this read endpoint
// works unauthenticated regardless of what's sent as uid.
func ResolveByNumberID(numberID string) (*PlayerRef, error) {
	reqBody, err := msgpack.Marshal(map[string]any{
		"uid":          randomUID(),
		"number_id":    numberID,
		"force_search": false,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, findPeopleByNumberIDURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var parsed struct {
		Result struct {
			ID      string `msgpack:"id"`
			Hostnum int    `msgpack:"hostnum"`
			Base    struct {
				Nickname string `msgpack:"nickname"`
			} `msgpack:"base"`
		} `msgpack:"result"`
	}
	if err := msgpack.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if parsed.Result.ID == "" {
		return nil, ErrNotFound
	}
	return &PlayerRef{
		PID:      parsed.Result.ID,
		Hostnum:  parsed.Result.Hostnum,
		Nickname: parsed.Result.Base.Nickname,
	}, nil
}

func readBody(resp *http.Response) ([]byte, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		return io.ReadAll(gz)
	}
	return io.ReadAll(resp.Body)
}

// randomUID is a client-generated per-request nonce, not a real identity —
// NetEase's own client sends a fresh one on every request (see
// wbm-relay/PLAN.md's "the token is portable" finding); any value works.
func randomUID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
