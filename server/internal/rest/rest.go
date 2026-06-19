// Package rest is the HTTP delivery adapter (ADR-0006 hexagonal core). It
// translates the OpenAPI contract in api/openapi.yaml into calls on the
// application services and maps errors to RFC 9457 problem details.
package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// New builds the HTTP handler for the whole API surface.
func New(authsvc *auth.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("POST /auth/register", handleRegister(authsvc))
	mux.HandleFunc("POST /auth/sessions", handleSignIn(authsvc))
	return mux
}

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

func newSessionView(actor domain.Actor, token string) sessionView {
	return sessionView{
		Token: token,
		Actor: actorView{
			ID:       string(actor.ID),
			Handle:   actor.Handle,
			Instance: actor.Instance.Domain,
			Local:    actor.Local,
		},
	}
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
