package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Guild struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Builders      []string `json:"builders"`
	Tags          []string `json:"tags,omitempty"`
	DiscordThread string   `json:"discordThread"`
	Lore          string   `json:"lore,omitempty"`
	WhatToVisit   string   `json:"whatToVisit,omitempty"`
	Score         int      `json:"score"`
	Screenshots   []string `json:"screenshots,omitempty"`
}

func loadGuilds(root string) ([]Guild, error) {
	data, err := os.ReadFile(filepath.Join(root, "guilds.json"))
	if err != nil {
		return nil, err
	}
	var guilds []Guild
	return guilds, json.Unmarshal(data, &guilds)
}

func saveGuilds(root string, guilds []Guild) error {
	data, err := json.MarshalIndent(guilds, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "guilds.json"), data, 0644)
}
