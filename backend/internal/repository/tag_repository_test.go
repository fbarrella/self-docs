package repository

import (
	"context"
	"testing"

	"github.com/self-docs/backend/internal/model"
)

func TestNormalizeAndCanonicalTag(t *testing.T) {
	tests := []struct {
		in        string
		norm      string
		canonical string
	}{
		{"Git", "git", "Git"},
		{"  #CLI  ", "cli", "CLI"},
		{"Front End", "front end", "Front End"},
		{"", "", ""},
	}
	for _, tt := range tests {
		if got := NormalizeTag(tt.in); got != tt.norm {
			t.Errorf("NormalizeTag(%q) = %q, want %q", tt.in, got, tt.norm)
		}
		if got := CanonicalTagName(tt.in); got != tt.canonical {
			t.Errorf("CanonicalTagName(%q) = %q, want %q", tt.in, got, tt.canonical)
		}
	}
}

func TestTagUpsertIsIdempotent(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewTagRepository(pool)

	first, err := repo.Upsert(ctx, "Git")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	second, err := repo.Upsert(ctx, "#git")
	if err != nil {
		t.Fatalf("Upsert again: %v", err)
	}
	if first.Name != second.Name {
		t.Errorf("names differ: %q vs %q", first.Name, second.Name)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM tags`).Scan(&count); err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if count != 1 {
		t.Errorf("tag count = %d, want 1", count)
	}
}

func TestTagPopularAndPrivateExclusion(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	docs := NewDocumentRepository(pool)
	tags := NewTagRepository(pool)

	for i := 0; i < 3; i++ {
		if _, err := docs.Create(ctx, CreateDocumentInput{
			Title: "Public " + string(rune('A'+i)), Section: model.SectionWorkflow, Tags: []string{"shared"},
		}); err != nil {
			t.Fatalf("create public: %v", err)
		}
	}
	if _, err := docs.Create(ctx, CreateDocumentInput{
		Title: "Private", Section: model.SectionPrivate, Tags: []string{"shared", "secret"},
	}); err != nil {
		t.Fatalf("create private: %v", err)
	}

	popular, err := tags.Popular(ctx, 10)
	if err != nil {
		t.Fatalf("Popular: %v", err)
	}
	var sharedCount int
	for _, tag := range popular {
		if tag.Name == "shared" {
			sharedCount = tag.Count
		}
		if tag.Name == "secret" {
			t.Errorf("private-only tag leaked: %+v", tag)
		}
	}
	if sharedCount != 3 {
		t.Errorf("shared count = %d, want 3 (private excluded)", sharedCount)
	}

	list, _, err := tags.List(ctx, ListTagsOptions{Sort: "count", Order: "desc"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, tag := range list {
		if tag.Name == "secret" {
			t.Errorf("private-only tag leaked into list: %+v", tag)
		}
	}
}

func TestTagDelete(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewTagRepository(pool)

	if _, err := repo.Upsert(ctx, "temp"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := repo.Delete(ctx, "TEMP"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByNormalized(ctx, "temp"); err != ErrNotFound {
		t.Errorf("GetByNormalized after delete = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, "temp"); err != ErrNotFound {
		t.Errorf("second Delete = %v, want ErrNotFound", err)
	}
}

func TestTagListPaginationAndSearch(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewTagRepository(pool)

	for _, name := range []string{"alpha", "beta", "alphabet", "gamma"} {
		if _, err := repo.Upsert(ctx, name); err != nil {
			t.Fatalf("Upsert %s: %v", name, err)
		}
	}

	page, pag, err := repo.List(ctx, ListTagsOptions{Query: "alpha", Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pag.Total != 2 {
		t.Errorf("Total = %d, want 2", pag.Total)
	}
	if len(page) != 1 {
		t.Errorf("len(page) = %d, want 1", len(page))
	}
}
