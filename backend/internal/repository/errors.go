// Package repository isolates all SQL access behind interfaces used by the
// handler layer. Every query that can surface private documents enforces the
// is_private = false predicate in this package so no call site can forget it
// (data-model.md section 6).
package repository

import "errors"

// Sentinel errors returned by repositories and mapped to HTTP codes by handlers.
var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned on unique-violation (e.g. slug collision).
	ErrConflict = errors.New("conflict")
	// ErrInvalidHierarchy is returned when a parent is not a project_note or
	// would introduce a cycle.
	ErrInvalidHierarchy = errors.New("invalid hierarchy")
)
