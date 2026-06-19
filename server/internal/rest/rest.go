// Package rest is the HTTP delivery adapter (ADR-0006 hexagonal core). It
// translates the OpenAPI contract in api/openapi.yaml into calls on the
// application services and maps errors to RFC 9457 problem details.
//
// Routing and middleware use go-chi/chi: the stdlib ServeMux covers method and
// path routing, but chi gives ergonomic middleware and route grouping, which
// arrive with the first protected route (see docs/adr/0007).
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// New builds the HTTP handler for the whole API surface.
func New(authsvc *auth.Service) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", handleHealthz)
	r.Post("/auth/register", handleRegister(authsvc))
	r.Post("/auth/sessions", handleSignIn(authsvc))

	r.Group(func(r chi.Router) {
		r.Use(requireAuth(authsvc))
		r.Get("/me", handleMe)
	})

	return r
}

// ctxKey is the unexported request-context key type for this package.
type ctxKey int

const actorKey ctxKey = iota

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type actorView struct {
	ID       string `json:"id"`
	Handle   string `json:"handle"`
	Instance string `json:"instance"`
	Local    bool   `json:"local"`
}

type sessionView struct {
	Token string    `json:"token"`
	Actor actorView `json:"actor"`
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleRegister(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		creds, ok := decodeCredentials(w, r)
		if !ok {
			return
		}
		actor, token, err := svc.Register(r.Context(), creds.Email, creds.Password)
		switch {
		case err == nil:
			writeJSON(w, http.StatusCreated, newSessionView(actor, token))
		case errors.Is(err, auth.ErrEmailTaken):
			writeProblem(w, http.StatusConflict, "Email already registered", "An account already exists for this email.")
		case errors.Is(err, auth.ErrWeakPassword):
			writeProblem(w, http.StatusBadRequest, "Weak password", "The password is too short.")
		default:
			writeProblem(w, http.StatusInternalServerError, "Registration failed", "")
		}
	}
}

func handleSignIn(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		creds, ok := decodeCredentials(w, r)
		if !ok {
			return
		}
		actor, token, err := svc.Authenticate(r.Context(), creds.Email, creds.Password)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, newSessionView(actor, token))
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeProblem(w, http.StatusUnauthorized, "Authentication failed", "Unknown email or wrong password.")
		default:
			writeProblem(w, http.StatusInternalServerError, "Authentication failed", "")
		}
	}
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	actor, _ := r.Context().Value(actorKey).(domain.Actor)
	writeJSON(w, http.StatusOK, toActorView(actor))
}

// requireAuth is middleware that resolves the bearer token to an actor and
// stashes it in the request context, or replies 401.
func requireAuth(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeProblem(w, http.StatusUnauthorized, "Unauthorized", "A bearer token is required.")
				return
			}
			actor, err := svc.ActorFromToken(r.Context(), token)
			if err != nil {
				writeProblem(w, http.StatusUnauthorized, "Unauthorized", "The token is missing or invalid.")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey, actor)))
		})
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header. The scheme is matched case-insensitively per RFC 7235.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var c credentials
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request body", "Expected a JSON object with email and password.")
		return credentials{}, false
	}
	if c.Email == "" || c.Password == "" {
		writeProblem(w, http.StatusBadRequest, "Missing credentials", "Both email and password are required.")
		return credentials{}, false
	}
	return c, true
}

func toActorView(a domain.Actor) actorView {
	return actorView{
		ID:       string(a.ID),
		Handle:   a.Handle,
		Instance: a.Instance.Domain,
		Local:    a.Local,
	}
}

func newSessionView(actor domain.Actor, token string) sessionView {
	return sessionView{Token: token, Actor: toActorView(actor)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeProblem emits an RFC 9457 problem detail.
func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"title":  title,
		"status": status,
		"detail": detail,
	})
}
