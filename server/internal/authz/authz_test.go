package authz

import (
	"context"
	"testing"
)

func TestMemoryCheckWrite(t *testing.T) {
	ctx := context.Background()
	a := NewMemory()

	grant := Tuple{Subject: "user:ada", Relation: "editor", Object: "task:1"}

	if ok, _ := a.Check(ctx, grant); ok {
		t.Fatal("expected no access before any write")
	}

	if err := a.Write(ctx, []Tuple{grant}, nil); err != nil {
		t.Fatalf("write add: %v", err)
	}
	if ok, _ := a.Check(ctx, grant); !ok {
		t.Fatal("expected access after grant")
	}

	if err := a.Write(ctx, nil, []Tuple{grant}); err != nil {
		t.Fatalf("write remove: %v", err)
	}
	if ok, _ := a.Check(ctx, grant); ok {
		t.Fatal("expected no access after revoke")
	}
}
