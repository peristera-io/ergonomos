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
	"github.com/peristera-io/ergonomos/server/internal/authz"
	"github.com/peristera-io/ergonomos/server/internal/domain"
	"github.com/peristera-io/ergonomos/server/internal/events"
	"github.com/peristera-io/ergonomos/server/internal/rest"
	"github.com/peristera-io/ergonomos/server/internal/task"
)

// validPassword is the password the specs mean by "a valid password" / "the
// correct password". Sign-in with anything else is treated as incorrect.
const validPassword = "correct horse battery staple"

// world holds the per-scenario state: the running instance, the most recent
// token observed, and the last response seen through the API. Task scenarios
// add the caller's actor id, a second user's token, and the last task/list
// observed.
type world struct {
	server *httptest.Server
	status int
	body   map[string]any
	list   []any
	token  string

	myActorID  string
	otherToken string
	taskID     string
	myTitles   []string
}

func (w *world) aRunningInstance() error {
	instance := domain.Instance{ID: domain.NewID(), Domain: "localhost"}
	authsvc := auth.NewService(auth.NewMemoryStore(), auth.NewMemorySessions(), instance)
	tasksvc := task.NewService(task.NewMemoryStore(), authz.NewMemory(), events.NewMemoryOutbox())
	w.server = httptest.NewServer(rest.New(authsvc, tasksvc))
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
	return w.record(resp)
}

func (w *world) get(path, token string) error {
	req, err := http.NewRequest(http.MethodGet, w.server.URL+path, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return w.record(resp)
}

// record captures the status and decoded body of resp, remembering any token
// it carries for later authenticated requests.
func (w *world) record(resp *http.Response) error {
	defer resp.Body.Close()
	w.status = resp.StatusCode
	w.body = nil
	w.list = nil
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(data) > 0 {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return fmt.Errorf("decoding response body %q: %w", data, err)
		}
		switch decoded := v.(type) {
		case map[string]any:
			w.body = decoded
		case []any:
			w.list = decoded
		}
	}
	if t, ok := w.body["token"].(string); ok {
		w.token = t
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

// actorHandleIs reads the handle from the last response, accepting either a
// Session ({actor:{...}}, from register/sign-in) or a bare Actor (from /me).
func (w *world) actorHandleIs(want string) error {
	actor := w.body
	if nested, ok := w.body["actor"].(map[string]any); ok {
		actor = nested
	}
	got, _ := actor["handle"].(string)
	if got != want {
		return fmt.Errorf("actor handle = %q, want %q", got, want)
	}
	return nil
}

func (w *world) iHaveRegisteredAs(email string) error {
	if err := w.registerWithValidPassword(email); err != nil {
		return err
	}
	if w.status != http.StatusCreated {
		return fmt.Errorf("setup: registering %q returned status %d", email, w.status)
	}
	if actor, ok := w.body["actor"].(map[string]any); ok {
		w.myActorID, _ = actor["id"].(string)
	}
	return nil
}

func (w *world) requestProfileWithMyToken() error { return w.get("/me", w.token) }

func (w *world) requestProfileWithoutToken() error { return w.get("/me", "") }

func (w *world) requestProfileWithToken(token string) error { return w.get("/me", token) }

func (w *world) profileReturned() error {
	return w.statusShouldBe(http.StatusOK, "profile lookup")
}

func (w *world) requestUnauthorized() error {
	return w.statusShouldBe(http.StatusUnauthorized, "request")
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	w := &world{}

	ctx.Step(`^a running ergonomos instance$`, w.aRunningInstance)
	ctx.Step(`^I register with email "([^"]*)" and a valid password$`, w.registerWithValidPassword)
	ctx.Step(`^an account already exists for "([^"]*)"$`, w.accountExists)
	ctx.Step(`^an account exists for "([^"]*)"$`, w.accountExists)
	ctx.Step(`^I sign in with email "([^"]*)" and the correct password$`, w.signInCorrect)
	ctx.Step(`^I sign in with email "([^"]*)" and an incorrect password$`, w.signInIncorrect)

	ctx.Step(`^I have registered as "([^"]*)"$`, w.iHaveRegisteredAs)
	ctx.Step(`^I request my profile with my token$`, w.requestProfileWithMyToken)
	ctx.Step(`^I request my profile without a token$`, w.requestProfileWithoutToken)
	ctx.Step(`^I request my profile with the token "([^"]*)"$`, w.requestProfileWithToken)

	ctx.Step(`^my account is created$`, w.accountCreated)
	ctx.Step(`^I receive an authentication token$`, w.receiveToken)
	ctx.Step(`^my actor handle is "([^"]*)" on this instance$`, w.actorHandleIs)
	ctx.Step(`^my profile is returned$`, w.profileReturned)
	ctx.Step(`^registration is rejected as a conflict$`, w.registrationIsConflict)
	ctx.Step(`^authentication is rejected$`, w.authenticationRejected)
	ctx.Step(`^the request is unauthorized$`, w.requestUnauthorized)

	registerTaskSteps(ctx, w)

	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if w.server != nil {
			w.server.Close()
			w.server = nil
		}
		return ctx, nil
	})
}
