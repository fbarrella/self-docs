package cache

import "context"

// Key names used by the application. Centralized so invalidation and reads
// cannot drift apart.
const (
	KeyDashboard   = "dashboard"
	KeyPopularTags = "tags:popular"
)

// DocumentMutationKeys are invalidated whenever a document is created,
// updated, or deleted (and on imports).
var DocumentMutationKeys = []string{KeyDashboard, KeyPopularTags}

// InvalidateDocuments clears every cache entry affected by a document write.
func (c *Cache) InvalidateDocuments(ctx context.Context) {
	c.Delete(ctx, DocumentMutationKeys...)
}
