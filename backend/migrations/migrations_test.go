package migrations

import "testing"

func TestLoadFindsInitMigration(t *testing.T) {
	migs, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(migs) == 0 {
		t.Fatal("Load() returned no migrations")
	}
	if migs[0].Version != 1 || migs[0].Name != "init" {
		t.Errorf("first migration = %d/%s, want 1/init", migs[0].Version, migs[0].Name)
	}
	if migs[0].SQL == "" {
		t.Error("migration SQL is empty")
	}
}

func TestLoadSortedByVersion(t *testing.T) {
	migs, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	for i := 1; i < len(migs); i++ {
		if migs[i-1].Version >= migs[i].Version {
			t.Fatalf("migrations not strictly ascending: %d then %d", migs[i-1].Version, migs[i].Version)
		}
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		filename string
		version  int
		name     string
		wantErr  bool
	}{
		{"0001_init.up.sql", 1, "init", false},
		{"0002_add_assets.up.sql", 2, "add_assets", false},
		{"bad.up.sql", 0, "", true},
		{"abc_init.up.sql", 0, "", true},
	}
	for _, tt := range tests {
		version, name, err := parseName(tt.filename)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseName(%q) expected error", tt.filename)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseName(%q) unexpected error: %v", tt.filename, err)
			continue
		}
		if version != tt.version || name != tt.name {
			t.Errorf("parseName(%q) = %d/%s, want %d/%s", tt.filename, version, name, tt.version, tt.name)
		}
	}
}
