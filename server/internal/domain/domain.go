// Package domain holds the core entity types shared across ergonomos.
//
// These types encode the federation-readiness constraints from ADR-0004:
// every shareable entity carries a globally-unique opaque ID and a home
// Instance, so a future server-to-server protocol (likely ActivityPub) is not
// precluded.
package domain

import (
	"crypto/rand"
	"sync"

	"github.com/oklog/ulid/v2"
)

// ID is an opaque, globally-unique identifier in canonical ULID string form
// (Crockford base32, 26 chars). ULIDs are lexicographically sortable by
// creation time, which keeps database index locality good without exposing a
// sequential integer key (ADR-0004).
type ID string

// entropy is a monotonic ULID entropy source seeded from crypto/rand. The
// monotonic reader is not safe for concurrent use, so NewID guards it.
var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

// NewID returns a fresh ULID. Within the same millisecond, successive IDs are
// strictly increasing; across milliseconds they sort by time.
func NewID() ID {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ID(ulid.MustNew(ulid.Now(), entropy).String())
}

// Instance identifies an ergonomos server ("home instance") in a federation.
type Instance struct {
	ID     ID     // local opaque id
	Domain string // e.g. "tasks.example.org"
}

// Actor is any principal that can hold relationships to objects: a local user
// or a remote (federated) user. Remote actors do not authenticate locally.
//
// Handle is "ada" for a local actor and "ada@other.example" for a remote one.
type Actor struct {
	ID       ID
	Handle   string
	Instance Instance
	Local    bool
}
