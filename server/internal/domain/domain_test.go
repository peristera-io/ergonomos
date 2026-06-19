package domain

import (
	"regexp"
	"sort"
	"testing"
)

// Canonical ULID: 26 chars of Crockford base32 (excludes I, L, O, U).
var ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

func TestNewIDFormatAndUniqueness(t *testing.T) {
	const n = 1000
	ids := make([]string, 0, n)
	seen := make(map[ID]struct{}, n)
	for range n {
		id := NewID()
		if !ulidPattern.MatchString(string(id)) {
			t.Fatalf("NewID() = %q, not a canonical ULID", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("NewID() returned a duplicate: %q", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, string(id))
	}

	// IDs minted in sequence must already be in sorted order — that's the
	// index-locality property we adopted ULIDs for.
	if !sort.StringsAreSorted(ids) {
		t.Error("IDs generated in sequence are not lexicographically sorted")
	}
}
