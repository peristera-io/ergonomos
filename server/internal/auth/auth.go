// Package auth is the authentication boundary: registration, sign-in, and
// resolving a bearer token back to its actor. It depends only on ports (the
// Store and Sessions interfaces, ADR-0006 hexagonal core); the v1 adapters are
// in-memory, with Postgres adapters to follow behind the same interfaces.
//
// Third-party / OAuth2 authorization (ADR-0006) is still out of scope: tokens
// here are opaque session tokens for the interactive user.
package auth

import (
	"context"
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
	// ErrInvalidToken is returned by Sessions.Resolve for an unknown token.
	ErrInvalidToken = errors.New("auth: invalid token")
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

// Sessions issues opaque bearer tokens and resolves them back to an actor.
type Sessions interface {
	// Issue mints a fresh token bound to actor.
	Issue(ctx context.Context, actor domain.Actor) (string, error)
	// Resolve returns the actor a token was issued for, or ErrInvalidToken.
	Resolve(ctx context.Context, token string) (domain.Actor, error)
}

// Service performs registration, authentication, and token resolution.
type Service struct {
	store    Store
	sessions Sessions
	instance domain.Instance
}

// NewService builds a Service. instance is the home instance new local actors
// are minted on.
func NewService(store Store, sessions Sessions, instance domain.Instance) *Service {
	return &Service{store: store, sessions: sessions, instance: instance}
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
	token, err := s.sessions.Issue(ctx, actor)
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
	token, err := s.sessions.Issue(ctx, u.Actor)
	if err != nil {
		return domain.Actor{}, "", err
	}
	return u.Actor, token, nil
}

// ActorFromToken resolves a bearer token to its actor, returning
// ErrInvalidToken if the token is unknown.
func (s *Service) ActorFromToken(ctx context.Context, token string) (domain.Actor, error) {
	return s.sessions.Resolve(ctx, token)
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
