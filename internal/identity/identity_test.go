package identity

import (
	"path/filepath"
	"testing"
)

func TestLoadCreatesAndPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity.json")

	first, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if first.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if first.Name == "" {
		t.Fatal("expected non-empty default Name")
	}

	second, err := Load(path)
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("id changed across reload: %q != %q", first.ID, second.ID)
	}
	if second.Name != first.Name {
		t.Fatalf("name changed across reload: %q != %q", first.Name, second.Name)
	}
}

func TestSetNamePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity.json")

	id, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := id.SetName("Marek's MacBook"); err != nil {
		t.Fatalf("SetName: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Name != "Marek's MacBook" {
		t.Fatalf("got name %q, want %q", reloaded.Name, "Marek's MacBook")
	}
}
