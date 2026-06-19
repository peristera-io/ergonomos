// Package auth is the authentication boundary: registration and sign-in for
// local accounts. It depends only on the Store port (ADR-0006 hexagonal core);
// the v1 adapter is in-memory, with a Postgres adapter to follow behind the
// same interface.
//
// Token persistence and validation (and third-party / OAuth2 authorization,
// ADR-0006) are deliberately out of this first slice: tokens are opaque and
// minted here, but there are no protected routes consuming them yet.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// Service-level errors. Authenticate collapses "unknown email" and "wrong
// password" into ErrInvalidCredentials so the API cannot be used to enumerate
// accounts.
var (
	ErrEmailTaken         = errors.New("auth: an account already exists for this email")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrWeakPassword       = errors.New("auth: password too short")
	// ErrNotFound is returned by a Store when no user matches.
	ErrNotFound = errors.New("auth: user not found")
)

// MinPasswordLen is the shortest password we accept at registration.
const MinPasswordLen = 8

// User is a registered local account and its stored credentials.
type User struct {
	Actor        domain.Actor
	Email        string
	PasswordHash string
}

// Store persists user accounts.
type Store interface {
	// CreateUser stores u, returning ErrEmailTaken if the email is in use.
	CreateUser(ctx context.Context, u User) error
	// FindByEmail returns the user for email, or ErrNotFound if none exists.
	FindByEmail(ctx context.Context, email string) (User, error)
}

// Service performs registration and authentication against a Store.
type Service struct {
	store    Store
	instance domain.Instance
}

// NewService builds a Service. instance is the home instance new local actors
// are minted on.
func NewService(store Store, instance domain.Instance) *Service {
	return &Service{store: store, instance: instance}
}

// Register creates a new local account and returns the actor with a fresh
// authentication token.
func (s *Service) Register(ctx context.Context, email, password string) (domain.Actor, string, error) {
	email = normalizeEmail(email)
	if len(password) < MinPasswordLen {
		return domain.Actor{}, "", ErrWeakPassword
	}
	hash, err := hashPassword(password)
	if err != nil {
		return domain.Actor{}, "", err
	}
	actor := domain.Actor{
		ID:       domain.NewID(),
		Handle:   handleFromEmail(email),
		Instance: s.instance,
		Local:    true,
	}
	if err := s.store.CreateUser(ctx, User{Actor: actor, Email: email, PasswordHash: hash}); err != nil {
		return domain.Actor{}, "", err
	}
	token, err := newToken()
	if err != nil {
		return domain.Actor{}, "", err
	}
	return actor, token, nil
}

// Authenticate verifies credentials and returns the actor with a fresh token.
func (s *Service) Authenticate(ctx context.Context, email, password string) (domain.Actor, string, error) {
	email = normalizeEmail(email)
	u, err := s.store.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.Actor{}, "", ErrInvalidCredentials
		}
		return domain.Actor{}, "", err
	}
	if !verifyPassword(u.PasswordHash, password) {
		return domain.Actor{}, "", ErrInvalidCredentials
	}
	token, err := newToken()
	if err != nil {
		return domain.Actor{}, "", err
	}
	return u.Actor, token, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// handleFromEmail derives a local actor handle from the email local-part:
// "ada@example.org" -> "ada".
func handleFromEmail(email string) string {
	if i := strings.IndexByte(email, '@'); i >= 0 {
		return email[:i]
	}
	return email
}

// newToken mints an opaque 256-bit bearer token.
func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
