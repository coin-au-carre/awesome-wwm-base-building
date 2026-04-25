package discord

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CatalogItem holds the fields needed to display a catalog item.
type CatalogItem struct {
	Name        string
	Filename    string
	Category    string
	SubCategory string
}

// SearchCatalogItems returns items from the catalog whose name or subcategory
// contains the query (case-insensitive). Results are capped at maxResults.
func SearchCatalogItems(root, query string, maxResults int) []CatalogItem {
	data, err := os.ReadFile(filepath.Join(root, "catalog", "guild", "guild_items.json"))
	if err != nil {
		return nil
	}
	var catalog map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil
	}

	q := strings.ToLower(query)
	var results []CatalogItem
	for cat, cv := range catalog {
		for subCat, raw := range cv {
			if subCat == "translations" {
				continue
			}
			var sub struct {
				Items []struct {
					Name     string `json:"name"`
					Filename string `json:"filename"`
				} `json:"items"`
			}
			if err := json.Unmarshal(raw, &sub); err != nil {
				continue
			}
			subMatch := strings.Contains(strings.ToLower(subCat), q)
			for _, it := range sub.Items {
				if subMatch || strings.Contains(strings.ToLower(it.Name), q) {
					results = append(results, CatalogItem{
						Name:        it.Name,
						Filename:    it.Filename,
						Category:    cat,
						SubCategory: subCat,
					})
					if len(results) >= maxResults {
						return results
					}
				}
			}
		}
	}
	return results
}

// CatalogImagePath returns the local filesystem path for a catalog item image.
func CatalogImagePath(root string, item CatalogItem) string {
	return filepath.Join(root, "catalog", "guild", item.Category, item.SubCategory, item.Filename)
}
