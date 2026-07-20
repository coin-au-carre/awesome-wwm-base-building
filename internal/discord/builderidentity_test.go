package discord

import (
	"os"
	"testing"
)

func TestFindBuilderIdentityByDiscordID(t *testing.T) {
	identities := []BuilderIdentity{
		{DiscordID: "1", CanonicalSlug: "alice"},
		{DiscordID: "2", CanonicalSlug: "bob"},
	}

	if idx := FindBuilderIdentityByDiscordID(identities, "2"); idx != 1 {
		t.Errorf("got index %d, want 1", idx)
	}
	if idx := FindBuilderIdentityByDiscordID(identities, "missing"); idx != -1 {
		t.Errorf("got index %d, want -1", idx)
	}
}

func TestFindBuilderIdentityBySlug(t *testing.T) {
	identities := []BuilderIdentity{
		{DiscordID: "1", CanonicalSlug: "alice"},
		{DiscordID: "2", CanonicalSlug: "bob"},
	}

	if idx := FindBuilderIdentityBySlug(identities, "bob"); idx != 1 {
		t.Errorf("got index %d, want 1", idx)
	}
	if idx := FindBuilderIdentityBySlug(identities, "carol"); idx != -1 {
		t.Errorf("got index %d, want -1", idx)
	}
}

func TestBuilderIdentitiesRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root+"/data", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	want := []BuilderIdentity{
		{DiscordID: "1", CanonicalAlias: "Alice", CanonicalSlug: "alice", Aliases: []string{"al"}, NeteaseNumberID: "123", NeteasePID: "aXYZ", NeteaseHostnum: 10203, IngameNickname: "Alice"},
	}
	if err := SaveBuilderIdentities(root, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := LoadBuilderIdentities(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].CanonicalSlug != "alice" || got[0].NeteaseHostnum != 10203 {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestLoadBuilderIdentitiesMissingFile(t *testing.T) {
	root := t.TempDir()
	got, err := LoadBuilderIdentities(root)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}
