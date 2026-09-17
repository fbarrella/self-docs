package repository

import (
	"context"
	"testing"

	"github.com/self-docs/backend/internal/model"
)

func TestDocumentCreateAndGet(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	doc, err := repo.Create(ctx, CreateDocumentInput{
		Title:   "Git Commands",
		Content: "# Git Commands\n\nA reference.",
		Excerpt: strptr("Reference for common Git commands."),
		Section: model.SectionCheatSheet,
		Tags:    []string{"Git", "CLI"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if doc.ID == "" {
		t.Error("expected generated id")
	}
	if doc.Slug != "git-commands" {
		t.Errorf("Slug = %q, want git-commands", doc.Slug)
	}
	if len(doc.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 entries", doc.Tags)
	}

	got, err := repo.GetByID(ctx, doc.ID, false)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != doc.Title || got.Content != doc.Content {
		t.Errorf("GetByID mismatch: %+v", got)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %v, want 2", got.Tags)
	}
}

func TestDocumentSlugConflict(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	in := CreateDocumentInput{Title: "Same Title", Section: model.SectionWorkflow}
	if _, err := repo.Create(ctx, in); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if _, err := repo.Create(ctx, in); err != ErrConflict {
		t.Fatalf("second Create error = %v, want ErrConflict", err)
	}
}

func TestDocumentProjectNoteHierarchy(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	root, err := repo.Create(ctx, CreateDocumentInput{Title: "Frontend", Section: model.SectionProjectNote})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	child, err := repo.Create(ctx, CreateDocumentInput{
		Title: "Routing", Section: model.SectionProjectNote, ParentID: &root.ID,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	tree, err := repo.Tree(ctx, model.SectionProjectNote, false)
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("tree roots = %d, want 1", len(tree))
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].ID != child.ID {
		t.Errorf("unexpected tree shape: %+v", tree[0])
	}

	if _, err := repo.Create(ctx, CreateDocumentInput{
		Title: "Bad", Section: model.SectionWorkflow, ParentID: &root.ID,
	}); err == nil {
		t.Error("expected error creating nested non-project-note")
	}
}

func TestDocumentCycleRejected(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	root, _ := repo.Create(ctx, CreateDocumentInput{Title: "Root", Section: model.SectionProjectNote})
	child, _ := repo.Create(ctx, CreateDocumentInput{Title: "Child", Section: model.SectionProjectNote, ParentID: &root.ID})

	if _, err := repo.Update(ctx, root.ID, UpdateDocumentInput{
		SetParent: true, ParentID: &child.ID,
	}, false); err == nil {
		t.Error("expected cycle to be rejected")
	}
}

func TestDocumentListFiltersAndPagination(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	for i := 0; i < 5; i++ {
		title := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}[i]
		section := model.SectionWorkflow
		if i%2 == 0 {
			section = model.SectionCheatSheet
		}
		if _, err := repo.Create(ctx, CreateDocumentInput{
			Title: title, Section: section, Tags: []string{"shared", title},
		}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}

	docs, pag, err := repo.List(ctx, ListDocumentsOptions{
		Sections: []model.Section{model.SectionCheatSheet},
		Page:     1, PageSize: 2,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pag.Total != 3 {
		t.Errorf("Total = %d, want 3", pag.Total)
	}
	if len(docs) != 2 {
		t.Errorf("len(docs) = %d, want 2", len(docs))
	}
	for _, d := range docs {
		if d.Section != model.SectionCheatSheet {
			t.Errorf("section = %q, want cheat_sheet", d.Section)
		}
	}

	tagged, _, err := repo.List(ctx, ListDocumentsOptions{Tags: []string{"shared"}})
	if err != nil {
		t.Fatalf("List by tag: %v", err)
	}
	if len(tagged) != 5 {
		t.Errorf("tagged count = %d, want 5", len(tagged))
	}
}

func TestDocumentPrivateExclusion(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	pub, _ := repo.Create(ctx, CreateDocumentInput{Title: "Public", Section: model.SectionWorkflow})
	priv, _ := repo.Create(ctx, CreateDocumentInput{Title: "Secret", Section: model.SectionPrivate})

	list, _, err := repo.List(ctx, ListDocumentsOptions{IncludePrivate: false})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, d := range list {
		if d.ID == priv.ID {
			t.Error("private document leaked into public list")
		}
	}

	if _, err := repo.GetByID(ctx, priv.ID, false); err != ErrNotFound {
		t.Errorf("GetByID(private, false) error = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByID(ctx, priv.ID, true); err != nil {
		t.Errorf("GetByID(private, true) error = %v, want nil", err)
	}
	_ = pub

	withPrivate, _, _ := repo.List(ctx, ListDocumentsOptions{IncludePrivate: true})
	if len(withPrivate) != 2 {
		t.Errorf("IncludePrivate list = %d, want 2", len(withPrivate))
	}
}

func TestDocumentUpdateAndTags(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	doc, _ := repo.Create(ctx, CreateDocumentInput{
		Title: "Original", Content: "v1", Section: model.SectionWorkflow, Tags: []string{"old"},
	})

	newTitle := "Renamed"
	newTags := []string{"new", "fresh"}
	updated, err := repo.Update(ctx, doc.ID, UpdateDocumentInput{
		Title: &newTitle, Tags: &newTags,
	}, false)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != newTitle || updated.Slug != "renamed" {
		t.Errorf("title/slug = %q/%q", updated.Title, updated.Slug)
	}
	if len(updated.Tags) != 2 {
		t.Errorf("tags = %v, want 2", updated.Tags)
	}
	for _, tag := range updated.Tags {
		if tag == "old" {
			t.Errorf("stale tag still attached: %v", updated.Tags)
		}
	}

	reloaded, _ := repo.GetByID(ctx, doc.ID, false)
	for _, tag := range reloaded.Tags {
		if tag == "old" {
			t.Errorf("stale tag still attached after reload: %v", reloaded.Tags)
		}
	}
}

func TestDocumentDelete(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	doc, _ := repo.Create(ctx, CreateDocumentInput{Title: "Temp", Section: model.SectionWorkflow})
	if err := repo.Delete(ctx, doc.ID, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, doc.ID, false); err != ErrNotFound {
		t.Errorf("GetByID after delete = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, doc.ID, false); err != ErrNotFound {
		t.Errorf("second Delete = %v, want ErrNotFound", err)
	}
}

func TestDocumentSearchExcludesPrivate(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	if _, err := repo.Create(ctx, CreateDocumentInput{
		Title: "Git Rebase Guide", Content: "How to rebase and cherry-pick", Section: model.SectionWorkflow,
	}); err != nil {
		t.Fatalf("create public: %v", err)
	}
	if _, err := repo.Create(ctx, CreateDocumentInput{
		Title: "Secret Rebase Notes", Content: "private rebase details", Section: model.SectionPrivate,
	}); err != nil {
		t.Fatalf("create private: %v", err)
	}

	results, pag, err := repo.Search(ctx, SearchOptions{Query: "rebase"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if pag.Total != 1 || len(results) != 1 {
		t.Fatalf("total=%d len=%d, want 1/1 (private must be excluded)", pag.Total, len(results))
	}
	if results[0].Title != "Git Rebase Guide" {
		t.Errorf("unexpected result: %+v", results[0])
	}
	if results[0].Snippet == "" {
		t.Error("expected a non-empty snippet")
	}

	privateOnly, _, err := repo.Search(ctx, SearchOptions{Query: "cherry-pick"})
	if err != nil {
		t.Fatalf("Search private-only term: %v", err)
	}
	if len(privateOnly) != 1 {
		t.Errorf("cherry-pick results = %d, want 1", len(privateOnly))
	}

	secret, _, _ := repo.Search(ctx, SearchOptions{Query: "private"})
	if len(secret) != 0 {
		t.Errorf("search for private-only content returned %d results", len(secret))
	}
}

func TestDocumentCountsAndRecent(t *testing.T) {
	pool := testPool(t)
	truncate(t, pool)
	ctx := context.Background()
	repo := NewDocumentRepository(pool)

	for _, in := range []CreateDocumentInput{
		{Title: "W1", Section: model.SectionWorkflow},
		{Title: "W2", Section: model.SectionWorkflow},
		{Title: "S1", Section: model.SectionCheatSheet},
		{Title: "P1", Section: model.SectionPrivate},
	} {
		if _, err := repo.Create(ctx, in); err != nil {
			t.Fatalf("create %s: %v", in.Title, err)
		}
	}

	counts, err := repo.CountBySection(ctx, false)
	if err != nil {
		t.Fatalf("CountBySection: %v", err)
	}
	if counts[model.SectionWorkflow] != 2 || counts[model.SectionCheatSheet] != 1 {
		t.Errorf("counts = %v", counts)
	}
	if _, ok := counts[model.SectionPrivate]; ok {
		t.Errorf("private count leaked: %v", counts)
	}

	countsPrivate, _ := repo.CountBySection(ctx, true)
	if countsPrivate[model.SectionPrivate] != 1 {
		t.Errorf("private count = %d, want 1", countsPrivate[model.SectionPrivate])
	}

	recent, err := repo.RecentlyUpdated(ctx, 10, false)
	if err != nil {
		t.Fatalf("RecentlyUpdated: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("recent = %d, want 3 (private excluded)", len(recent))
	}
	for _, d := range recent {
		if d.Section == model.SectionPrivate {
			t.Error("private document in recently updated")
		}
	}
}
