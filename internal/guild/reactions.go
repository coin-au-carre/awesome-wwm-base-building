package guild

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReactionMap maps Discord thread ID → emoji → list of voter user IDs.
type ReactionMap map[string]map[string][]string

// UserInfo holds a Discord user's global username and server nickname.
type UserInfo struct {
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
}

// DisplayName returns the server nickname if set, otherwise the global username.
func (u UserInfo) DisplayName() string {
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}

// UserMap maps Discord user ID → UserInfo.
type UserMap map[string]UserInfo

const (
	reactionsFilename = "data/reactions.json"
	usersFilename     = "data/users.json"
)

func LoadReactions(root string) (ReactionMap, error) {
	path := filepath.Join(root, reactionsFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(ReactionMap), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m ReactionMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return m, nil
}

func SaveReactions(root string, reactions ReactionMap) error {
	path := filepath.Join(root, reactionsFilename)
	data, err := json.MarshalIndent(reactions, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling reactions: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func LoadUsers(root string) (UserMap, error) {
	path := filepath.Join(root, usersFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(UserMap), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m UserMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return m, nil
}

func SaveUsers(root string, users UserMap) error {
	path := filepath.Join(root, usersFilename)
	data, err := json.MarshalIndent(users, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling users: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
