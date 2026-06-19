package features

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// registerTaskSteps binds the Gherkin steps of task.feature to w.
func registerTaskSteps(ctx *godog.ScenarioContext, w *world) {
	ctx.Step(`^I create a task with title "([^"]*)"$`, w.createMyTask)
	ctx.Step(`^I have created a task with title "([^"]*)"$`, w.haveCreatedTask)
	ctx.Step(`^I create a task without a token$`, w.createTaskWithoutToken)
	ctx.Step(`^another user "([^"]*)" has signed in$`, w.anotherUserSignedIn)
	ctx.Step(`^that user has created a task with title "([^"]*)"$`, w.thatUserCreatedTask)
	ctx.Step(`^I request that task by its ID$`, w.requestTaskByID)
	ctx.Step(`^I request that task without a token$`, w.requestTaskWithoutToken)
	ctx.Step(`^I request a task with an unknown ID$`, w.requestUnknownTask)
	ctx.Step(`^that user requests my task by its ID$`, w.thatUserRequestsMyTask)
	ctx.Step(`^I request my tasks$`, w.requestMyTasks)
	ctx.Step(`^I change that task's title to "([^"]*)"$`, w.changeTaskTitle)
	ctx.Step(`^I change that task's title to "([^"]*)" without a token$`, w.changeTaskTitleNoToken)
	ctx.Step(`^that user changes my task's title to "([^"]*)"$`, w.otherChangesTaskTitle)
	ctx.Step(`^I complete that task$`, w.completeTask)
	ctx.Step(`^that user completes my task$`, w.otherCompletesTask)
	ctx.Step(`^I delete that task$`, w.deleteTask)
	ctx.Step(`^I delete that task without a token$`, w.deleteTaskNoToken)
	ctx.Step(`^that user deletes my task$`, w.otherDeletesTask)
	ctx.Step(`^I delete a task with an unknown ID$`, w.deleteUnknownTask)

	ctx.Step(`^the task is created$`, w.taskIsCreated)
	ctx.Step(`^the task has an opaque, globally-unique identifier$`, w.taskHasOpaqueID)
	ctx.Step(`^the task owner is my actor$`, w.taskOwnerIsMyActor)
	ctx.Step(`^the request is rejected as invalid$`, w.requestRejectedAsInvalid)
	ctx.Step(`^the task is returned$`, w.taskIsReturned)
	ctx.Step(`^the task title is "([^"]*)"$`, w.taskTitleIs)
	ctx.Step(`^the task is not found$`, w.taskNotFound)
	ctx.Step(`^I receive a list containing both tasks$`, w.listContainsBothTasks)
	ctx.Step(`^I receive a list containing only my task "([^"]*)"$`, w.listContainsOnlyMyTask)
	ctx.Step(`^the task is marked done$`, w.taskIsMarkedDone)
	ctx.Step(`^the task is not marked done$`, w.taskIsNotMarkedDone)
	ctx.Step(`^the task is deleted$`, w.taskIsDeleted)
}

// postTask creates a task through the API using the given bearer token (empty
// for an unauthenticated request) and records the response.
func (w *world) postTask(token, title string) error {
	payload, err := json.Marshal(map[string]string{"title": title})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, w.server.URL+"/tasks", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return w.record(resp)
}

// patchTask updates a task's title through the API using the given bearer
// token (empty for an unauthenticated request) and records the response.
func (w *world) patchTask(token, id, title string) error {
	payload, err := json.Marshal(map[string]string{"title": title})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, w.server.URL+"/tasks/"+id, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return w.record(resp)
}

// completeTaskAs marks a task done through the API using the given bearer token.
func (w *world) completeTaskAs(token, id string) error {
	req, err := http.NewRequest(http.MethodPost, w.server.URL+"/tasks/"+id+"/complete", nil)
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

// deleteTaskAs removes a task through the API using the given bearer token
// (empty for an unauthenticated request) and records the response.
func (w *world) deleteTaskAs(token, id string) error {
	req, err := http.NewRequest(http.MethodDelete, w.server.URL+"/tasks/"+id, nil)
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

// registerUser registers a fresh account out-of-band, returning its token and
// actor id without disturbing the "last observed response" state.
func (w *world) registerUser(email string) (token, actorID string, err error) {
	payload, err := json.Marshal(map[string]string{"email": email, "password": validPassword})
	if err != nil {
		return "", "", err
	}
	resp, err := http.Post(w.server.URL+"/auth/register", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("registering %q returned status %d", email, resp.StatusCode)
	}
	var body struct {
		Token string `json:"token"`
		Actor struct {
			ID string `json:"id"`
		} `json:"actor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", err
	}
	return body.Token, body.Actor.ID, nil
}

func (w *world) createMyTask(title string) error {
	if err := w.postTask(w.token, title); err != nil {
		return err
	}
	if w.status == http.StatusCreated {
		w.taskID, _ = w.body["id"].(string)
		w.myTitles = append(w.myTitles, title)
	}
	return nil
}

func (w *world) haveCreatedTask(title string) error {
	if err := w.createMyTask(title); err != nil {
		return err
	}
	if w.status != http.StatusCreated {
		return fmt.Errorf("setup: creating task %q returned status %d", title, w.status)
	}
	return nil
}

func (w *world) createTaskWithoutToken() error {
	return w.postTask("", "Write acceptance tests")
}

func (w *world) anotherUserSignedIn(email string) error {
	token, _, err := w.registerUser(email)
	if err != nil {
		return err
	}
	w.otherToken = token
	return nil
}

func (w *world) thatUserCreatedTask(title string) error {
	return w.postTask(w.otherToken, title)
}

func (w *world) requestTaskByID() error { return w.get("/tasks/"+w.taskID, w.token) }

func (w *world) requestTaskWithoutToken() error { return w.get("/tasks/"+w.taskID, "") }

func (w *world) requestUnknownTask() error {
	return w.get("/tasks/"+string(domain.NewID()), w.token)
}

func (w *world) thatUserRequestsMyTask() error { return w.get("/tasks/"+w.taskID, w.otherToken) }

func (w *world) requestMyTasks() error { return w.get("/tasks", w.token) }

func (w *world) changeTaskTitle(title string) error { return w.patchTask(w.token, w.taskID, title) }

func (w *world) changeTaskTitleNoToken(title string) error {
	return w.patchTask("", w.taskID, title)
}

func (w *world) otherChangesTaskTitle(title string) error {
	return w.patchTask(w.otherToken, w.taskID, title)
}

func (w *world) completeTask() error { return w.completeTaskAs(w.token, w.taskID) }

func (w *world) otherCompletesTask() error { return w.completeTaskAs(w.otherToken, w.taskID) }

func (w *world) deleteTask() error { return w.deleteTaskAs(w.token, w.taskID) }

func (w *world) deleteTaskNoToken() error { return w.deleteTaskAs("", w.taskID) }

func (w *world) otherDeletesTask() error { return w.deleteTaskAs(w.otherToken, w.taskID) }

func (w *world) deleteUnknownTask() error { return w.deleteTaskAs(w.token, string(domain.NewID())) }

func (w *world) taskIsCreated() error {
	return w.statusShouldBe(http.StatusCreated, "task creation")
}

func (w *world) taskHasOpaqueID() error {
	id, _ := w.body["id"].(string)
	if len(id) != 26 {
		return fmt.Errorf("expected a 26-char ULID, got %q", id)
	}
	return nil
}

func (w *world) taskOwnerIsMyActor() error {
	owner, _ := w.body["owner"].(string)
	if owner != w.myActorID {
		return fmt.Errorf("task owner = %q, want my actor %q", owner, w.myActorID)
	}
	return nil
}

func (w *world) requestRejectedAsInvalid() error {
	return w.statusShouldBe(http.StatusBadRequest, "task creation")
}

func (w *world) taskIsReturned() error {
	return w.statusShouldBe(http.StatusOK, "task retrieval")
}

func (w *world) taskTitleIs(want string) error {
	got, _ := w.body["title"].(string)
	if got != want {
		return fmt.Errorf("task title = %q, want %q", got, want)
	}
	return nil
}

func (w *world) taskNotFound() error {
	return w.statusShouldBe(http.StatusNotFound, "task retrieval")
}

func (w *world) taskIsDeleted() error {
	return w.statusShouldBe(http.StatusNoContent, "task deletion")
}

func (w *world) taskIsMarkedDone() error {
	if done, _ := w.body["done"].(bool); !done {
		return fmt.Errorf("task done = %v, want true", w.body["done"])
	}
	return nil
}

func (w *world) taskIsNotMarkedDone() error {
	if done, _ := w.body["done"].(bool); done {
		return fmt.Errorf("task done = %v, want false", w.body["done"])
	}
	return nil
}

func (w *world) listTitles() ([]string, error) {
	titles := make([]string, 0, len(w.list))
	for _, item := range w.list {
		t, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("list item is not an object: %v", item)
		}
		title, _ := t["title"].(string)
		titles = append(titles, title)
	}
	return titles, nil
}

func (w *world) listContainsBothTasks() error {
	titles, err := w.listTitles()
	if err != nil {
		return err
	}
	if len(titles) != len(w.myTitles) {
		return fmt.Errorf("list has %d tasks %v, want %d %v", len(titles), titles, len(w.myTitles), w.myTitles)
	}
	for _, want := range w.myTitles {
		if !contains(titles, want) {
			return fmt.Errorf("list %v is missing %q", titles, want)
		}
	}
	return nil
}

func (w *world) listContainsOnlyMyTask(want string) error {
	titles, err := w.listTitles()
	if err != nil {
		return err
	}
	if len(titles) != 1 || titles[0] != want {
		return fmt.Errorf("list = %v, want exactly [%q]", titles, want)
	}
	return nil
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
