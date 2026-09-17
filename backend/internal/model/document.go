// Package model defines the domain types shared across the repository, service,
// and handler layers. Field names mirror the JSON contract in docs/api.md.
package model

import (
	"fmt"
	"time"
)

// Section is one of the four content pillars (PRD 3.2).
type Section string

const (
	SectionWorkflow    Section = "workflow"
	SectionProjectNote Section = "project_note"
	SectionCheatSheet  Section = "cheat_sheet"
	SectionPrivate     Section = "private"
)

// AllSections lists every valid section in display order.
var AllSections = []Section{
	SectionWorkflow,
	SectionProjectNote,
	SectionCheatSheet,
	SectionPrivate,
}

// Valid reports whether s is a known section.
func (s Section) Valid() bool {
	switch s {
	case SectionWorkflow, SectionProjectNote, SectionCheatSheet, SectionPrivate:
		return true
	default:
		return false
	}
}

// IsPrivate reports whether the section is the restricted Private Archive.
func (s Section) IsPrivate() bool {
	return s == SectionPrivate
}

// Label returns the human-readable section name used in UI copy.
func (s Section) Label() string {
	switch s {
	case SectionWorkflow:
		return "Workflows & Guides"
	case SectionProjectNote:
		return "Project Notes"
	case SectionCheatSheet:
		return "Cheat Sheets"
	case SectionPrivate:
		return "Private Archive"
	default:
		return string(s)
	}
}

// Document is a single knowledge-base entry. Content is omitted from list
// responses by the handler layer, not by this struct.
type Document struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Excerpt   *string   `json:"excerpt"`
	Section   Section   `json:"section"`
	ParentID  *string   `json:"parent_id"`
	Position  int       `json:"position"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DocumentNode is a recursive Project Notes tree node (api.md 4.6).
type DocumentNode struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Slug     string          `json:"slug"`
	Section  Section         `json:"section"`
	Position int             `json:"position"`
	Children []*DocumentNode `json:"children"`
}

// SearchResult is a ranked full-text search hit (api.md 6).
type SearchResult struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Section   Section   `json:"section"`
	Tags      []string  `json:"tags"`
	Snippet   string    `json:"snippet"`
	Rank      float64   `json:"rank"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Tag is a tag with an optional usage count (api.md 3).
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count,omitempty"`
}

// Action enumerates the recorded activity kinds (data-model.md 2).
type Action string

const (
	ActionCreated  Action = "created"
	ActionUpdated  Action = "updated"
	ActionDeleted  Action = "deleted"
	ActionImported Action = "imported"
	ActionUnlocked Action = "unlocked"
)

// Valid reports whether a is a known action.
func (a Action) Valid() bool {
	switch a {
	case ActionCreated, ActionUpdated, ActionDeleted, ActionImported, ActionUnlocked:
		return true
	default:
		return false
	}
}

// Activity is a single activity-feed entry (api.md 3).
type Activity struct {
	ID            string    `json:"id"`
	Action        Action    `json:"action"`
	DocumentID    *string   `json:"document_id"`
	DocumentTitle *string   `json:"document_title"`
	Section       *Section  `json:"section"`
	Actor         string    `json:"actor"`
	CreatedAt     time.Time `json:"created_at"`
}

// Pagination describes a page within a collection (api.md 3).
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPagination normalizes page/pageSize and computes the page count. It
// returns the sanitized values alongside the struct.
func NewPagination(page, pageSize, total int) (Pagination, int, int) {
	page, pageSize = NormalizePage(page), NormalizePageSize(pageSize)
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, page, pageSize
}

// DefaultPageSize and MaxPageSize bound collection requests.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// NormalizePage clamps a page number to at least 1.
func NormalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// NormalizePageSize clamps a page size to [1, MaxPageSize].
func NormalizePageSize(size int) int {
	switch {
	case size < 1:
		return DefaultPageSize
	case size > MaxPageSize:
		return MaxPageSize
	default:
		return size
	}
}

// Offset returns the SQL OFFSET for the given page and size.
func Offset(page, pageSize int) int {
	return (NormalizePage(page) - 1) * NormalizePageSize(pageSize)
}

// Validate checks invariants that must hold before persistence.
func (d *Document) Validate() error {
	if d.Title == "" {
		return fmt.Errorf("title must not be empty")
	}
	if len([]rune(d.Title)) > 300 {
		return fmt.Errorf("title must be at most 300 characters")
	}
	if !d.Section.Valid() {
		return fmt.Errorf("invalid section %q", d.Section)
	}
	if d.ParentID != nil && d.Section != SectionProjectNote {
		return fmt.Errorf("parent_id is only allowed for project_note documents")
	}
	return nil
}
