// Package domain holds the core entity types shared across ergonomos.
//
// These types encode the federation-readiness constraints from ADR-0004:
// every shareable entity carries a globally-unique opaque ID and a home
// Instance, so a future server-to-server protocol (likely ActivityPub) is not
// precluded.
package domain

// ID is an opaque, globally-unique identifier (ULID/UUID in canonical string
// form). We never expose sequential integer keys across the API boundary.
// Generation is wired in a later slice.
type ID string

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
