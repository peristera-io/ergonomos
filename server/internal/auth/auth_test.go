package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/peristera-io/ergonomos/server/internal/domain"
)

func newTestService() *Service {
	return NewService(NewMemoryStore(), domain.Instance{ID: domain.NewID(), Domain: "localhost"})
}

func TestRegister(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	actor, token, err := svc.Register(ctx, "Ada@Example.org", "correct horse battery")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if actor.Handle != "ada" {
		t.Errorf("handle = %q, want %q", actor.Handle, "ada")
	}
	if !actor.Local {
		t.Error("registered actor should be local")
	}
	if actor.Instance.Domain != "localhost" {
		t.Errorf("instance = %q, want localhost", actor.Instance.Domain)
	}
	if actor.ID == "" {
		t.Error("actor should have an id")
	}
	if token == "" {
		t.Error("expected a non-empty token")
	}
}

func TestRegisterRejectsDuplicate(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	if _, _, err := svc.Register(ctx, "ada@example.org", "correct horse battery"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	_, _, err := svc.Register(ctx, "ada@example.org", "another good password")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate Register err = %v, want ErrEmailTaken", err)
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	svc := newTestService()
	_, _, err := svc.Register(context.Background(), "ada@example.org", "short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("weak-password Register err = %v, want ErrWeakPassword", err)
	}
}

func TestAuthenticate(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	const pw = "correct horse battery"
	if _, _, err := svc.Register(ctx, "ada@example.org", pw); err != nil {
		t.Fatalf("Register: %v", err)
	}

	t.Run("correct password", func(t *testing.T) {
		actor, token, err := svc.Authenticate(ctx, "ada@example.org", pw)
		if err != nil {
			t.Fatalf("Authenticate: %v", err)
		}
		if actor.Handle != "ada" || token == "" {
			t.Errorf("got actor %+v token %q", actor, token)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		if _, _, err := svc.Authenticate(ctx, "ada@example.org", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unknown email", func(t *testing.T) {
		if _, _, err := svc.Authenticate(ctx, "nobody@example.org", pw); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := hashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !verifyPassword(hash, "correct horse battery") {
		t.Error("verifyPassword rejected the correct password")
	}
	if verifyPassword(hash, "wrong") {
		t.Error("verifyPassword accepted a wrong password")
	}
	if verifyPassword("garbage", "x") {
		t.Error("verifyPassword accepted a malformed hash")
	}
}
