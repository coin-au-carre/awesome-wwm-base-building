package discord

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BuilderIdentity is one canonical record per real builder — see
// docs/builder-identity.md. Mirrors web/src/lib/builder-aliases.ts's
// BuilderIdentity type; keep both in sync if the shape changes.
//
// CanonicalAlias/Aliases keep their natural display casing (any case is
// fine); CanonicalSlug is always slugify(CanonicalAlias) — lowercase, the
// actual public URL/matching identity, unique across every record.
type BuilderIdentity struct {
	DiscordID       string   `json:"discordId,omitempty"`
	CanonicalAlias  string   `json:"canonicalAlias"`
	CanonicalSlug   string   `json:"canonicalSlug"`
	Aliases         []string `json:"aliases,omitempty"`
	IngameNickname  string   `json:"ingameNickname,omitempty"`
	NeteaseNumberID string   `json:"neteaseNumberId,omitempty"`
	NeteasePID      string   `json:"neteasePid,omitempty"`
	NeteaseHostnum  int      `json:"neteaseHostnum,omitempty"`
}

func builderIdentitiesPath(root string) string {
	return filepath.Join(root, "data/builder_identities.json")
}

// LoadBuilderIdentities reads data/builder_identities.json. A missing file
// yields an empty slice; a corrupt file returns an error rather than
// silently discarding everyone's registered data.
func LoadBuilderIdentities(root string) ([]BuilderIdentity, error) {
	data, err := os.ReadFile(builderIdentitiesPath(root))
	if os.IsNotExist(err) {
		return []BuilderIdentity{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading builder_identities.json: %w", err)
	}
	var identities []BuilderIdentity
	if err := json.Unmarshal(data, &identities); err != nil {
		return nil, fmt.Errorf("parsing builder_identities.json: %w", err)
	}
	return identities, nil
}

func SaveBuilderIdentities(root string, identities []BuilderIdentity) error {
	data, err := json.MarshalIndent(identities, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling builder identities: %w", err)
	}
	if err := os.WriteFile(builderIdentitiesPath(root), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing builder_identities.json: %w", err)
	}
	return nil
}

// FindBuilderIdentityByDiscordID returns the index of discordID's record in
// identities, or -1 if they don't have one yet.
func FindBuilderIdentityByDiscordID(identities []BuilderIdentity, discordID string) int {
	for i, entry := range identities {
		if entry.DiscordID == discordID {
			return i
		}
	}
	return -1
}

// FindBuilderIdentityBySlug returns the index of the record whose
// CanonicalSlug matches slug, or -1 if none does.
func FindBuilderIdentityBySlug(identities []BuilderIdentity, slug string) int {
	for i, entry := range identities {
		if entry.CanonicalSlug == slug {
			return i
		}
	}
	return -1
}
