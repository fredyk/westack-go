package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry_BeforeAndAfterHooksFire(t *testing.T) {
	reg := NewRegistry[string]()
	var beforeFired, afterFired bool

	reg.Before(OpCreate, func(ctx context.Context, entity *string) error {
		beforeFired = true
		return nil
	})
	reg.After(OpCreate, func(ctx context.Context, entity *string) error {
		afterFired = true
		return nil
	})

	if err := reg.RunBefore(OpCreate, context.Background(), nil); err != nil {
		t.Fatalf("RunBefore() error = %v", err)
	}
	if err := reg.RunAfter(OpCreate, context.Background(), nil); err != nil {
		t.Fatalf("RunAfter() error = %v", err)
	}

	if !beforeFired {
		t.Error("before hook did not fire")
	}
	if !afterFired {
		t.Error("after hook did not fire")
	}
}

func TestRegistry_BeforeHookAbortsOperation(t *testing.T) {
	reg := NewRegistry[string]()
	abortErr := errors.New("validation failed")

	reg.Before(OpCreate, func(ctx context.Context, entity *string) error {
		return abortErr
	})

	err := reg.RunBefore(OpCreate, context.Background(), nil)
	if !errors.Is(err, abortErr) {
		t.Errorf("RunBefore() error = %v, want %v", err, abortErr)
	}
}

func TestRegistry_BeforeHookModifiesEntity(t *testing.T) {
	reg := NewRegistry[string]()

	reg.Before(OpCreate, func(ctx context.Context, entity *string) error {
		*entity = "modified"
		return nil
	})

	val := "original"
	if err := reg.RunBefore(OpCreate, context.Background(), &val); err != nil {
		t.Fatalf("RunBefore() error = %v", err)
	}
	if val != "modified" {
		t.Errorf("entity = %q, want %q", val, "modified")
	}
}

func TestRegistry_MultipleHooksRunInOrder(t *testing.T) {
	reg := NewRegistry[int]()
	var order []string

	reg.Before(OpUpdate, func(ctx context.Context, entity *int) error {
		order = append(order, "first")
		return nil
	})
	reg.Before(OpUpdate, func(ctx context.Context, entity *int) error {
		order = append(order, "second")
		return nil
	})

	if err := reg.RunBefore(OpUpdate, context.Background(), nil); err != nil {
		t.Fatalf("RunBefore() error = %v", err)
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("hook order = %v, want [first second]", order)
	}
}

func TestRegistry_NoHooksRegistered_IsNoOp(t *testing.T) {
	reg := NewRegistry[string]()

	if err := reg.RunBefore(OpDelete, context.Background(), nil); err != nil {
		t.Fatalf("RunBefore() on empty registry error = %v", err)
	}
	if err := reg.RunAfter(OpDelete, context.Background(), nil); err != nil {
		t.Fatalf("RunAfter() on empty registry error = %v", err)
	}
}

func TestRegistry_DifferentOperationsAreIndependent(t *testing.T) {
	reg := NewRegistry[string]()
	var createFired, deleteFired bool

	reg.Before(OpCreate, func(ctx context.Context, entity *string) error {
		createFired = true
		return nil
	})
	reg.Before(OpDelete, func(ctx context.Context, entity *string) error {
		deleteFired = true
		return nil
	})

	reg.RunBefore(OpCreate, context.Background(), nil)
	if !createFired {
		t.Error("create hook did not fire")
	}
	if deleteFired {
		t.Error("delete hook should not have fired")
	}
}
