package features

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/cucumber/godog"

	"github.com/peristera-io/ergonomos/server/internal/auth"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/rest"
)

// validPassword is the password the specs mean by "a valid password" / "the
// correct password". Sign-in with anything else is treated as incorrect.
const validPassword = "correct horse battery staple"

// world holds the per-scenario state: the running instance and the last
// response observed through the API.
type world struct {
	server *httptest.Server
	status int
	body   map[string]any
}

func (w *world) aRunningInstance() error {
	instance := domain.Instance{ID: domain.NewID(), Domain: "localhost"}
	svc := auth.NewService(auth.NewMemoryStore(), instance)
	w.server = httptest.NewServer(rest.New(svc))
	return nil
}

func (w *world) post(path, email, password string) error {
	payload, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		return err
	}
	resp, err := http.Post(w.server.URL+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w.status = resp.StatusCode
	w.body = nil
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &w.body); err != nil {
			return fmt.Errorf("decoding response body %q: %w", data, err)
		}
	}
	return nil
}

func (w *world) registerWithValidPassword(email string) error {
	return w.post("/auth/register", email, validPassword)
}

func (w *world) accountExists(email string) error {
	if err := w.registerWithValidPassword(email); err != nil {
		return err
	}
	if w.status != http.StatusCreated {
		return fmt.Errorf("setup: pre-registering %q returned status %d", email, w.status)
	}
	return nil
}

func (w *world) signInCorrect(email string) error {
	return w.post("/auth/sessions", email, validPassword)
}

func (w *world) signInIncorrect(email string) error {
	return w.post("/auth/sessions", email, "wrong-"+validPassword)
}

func (w *world) statusShouldBe(want int, what string) error {
	if w.status != want {
		return fmt.Errorf("%s: got status %d, want %d", what, w.status, want)
	}
	return nil
}

func (w *world) accountCreated() error {
	return w.statusShouldBe(http.StatusCreated, "account creation")
}

func (w *world) registrationIsConflict() error {
	return w.statusShouldBe(http.StatusConflict, "registration")
}

func (w *world) authenticationRejected() error {
	return w.statusShouldBe(http.StatusUnauthorized, "authentication")
}

func (w *world) receiveToken() error {
	token, _ := w.body["token"].(string)
	if token == "" {
		return fmt.Errorf("expected a non-empty token, got body %v", w.body)
	}
	return nil
}

func (w *world) actorHandleIs(want string) error {
	actor, _ := w.body["actor"].(map[string]any)
	got, _ := actor["handle"].(string)
	if got != want {
		return fmt.Errorf("actor handle = %q, want %q", got, want)
	}
	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	w := &world{}

	ctx.Step(`^a running ergonomos instance$`, w.aRunningInstance)
	ctx.Step(`^I register with email "([^"]*)" and a valid password$`, w.registerWithValidPassword)
	ctx.Step(`^an account already exists for "([^"]*)"$`, w.accountExists)
	ctx.Step(`^an account exists for "([^"]*)"$`, w.accountExists)
	ctx.Step(`^I sign in with email "([^"]*)" and the correct password$`, w.signInCorrect)
	ctx.Step(`^I sign in with email "([^"]*)" and an incorrect password$`, w.signInIncorrect)

	ctx.Step(`^my account is created$`, w.accountCreated)
	ctx.Step(`^I receive an authentication token$`, w.receiveToken)
	ctx.Step(`^my actor handle is "([^"]*)" on this instance$`, w.actorHandleIs)
	ctx.Step(`^registration is rejected as a conflict$`, w.registrationIsConflict)
	ctx.Step(`^authentication is rejected$`, w.authenticationRejected)

	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if w.server != nil {
			w.server.Close()
			w.server = nil
		}
		return ctx, nil
	})
}
