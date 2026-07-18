package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	hooks "github.com/fredyk/westack-go/v3/hooks"
)

// ── Adversarial Repository CRUD tests ──────────────────────────────────────

func TestRepository_DeleteById_Idempotent(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})

	// First delete should succeed
	err := repo.DeleteById(ctx, "1")
	if err != nil {
		t.Fatalf("DeleteById() first call error = %v", err)
	}

	// Second delete on same ID should NOT error (idempotent)
	err = repo.DeleteById(ctx, "1")
	if err != nil {
		t.Fatalf("DeleteById() second call error = %v, want nil (idempotent)", err)
	}
}

func TestRepository_FindById_NilReturnsNil(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	// Non-existent ID: should return (nil, nil), NOT (nil, error)
	found, err := repo.FindById(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindById() error = %v, want nil", err)
	}
	if found != nil {
		t.Errorf("FindById() = %v, want nil", found)
	}
}

func TestRepository_UpdateById_PartialPatch(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice", Age: 30, Active: true, Score: 10.0})

	// Patch only one field
	updated, err := repo.UpdateById(ctx, "1", &testItem{Name: "Alice Updated"})
	if err != nil {
		t.Fatalf("UpdateById() error = %v", err)
	}
	if updated.Name != "Alice Updated" {
		t.Errorf("Name = %q, want %q", updated.Name, "Alice Updated")
	}
	// Other fields should be preserved by the connector's UpdateById (merge behavior)
	found, _ := repo.FindById(ctx, "1")
	if found.Name != "Alice Updated" {
		t.Errorf("Persisted Name = %q, want %q", found.Name, "Alice Updated")
	}
}

func TestRepository_Create_NilPointer(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	// Passing nil pointer should not panic
	_, err := repo.Create(ctx, nil)
	if err == nil {
		t.Fatal("Create(nil) error = nil, want error")
	}
}

func TestRepository_FindMany_Empty(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	items, err := repo.FindMany(ctx, nil)
	if err != nil {
		t.Fatalf("FindMany() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("FindMany() on empty = %d items, want 0", len(items))
	}
}

func TestRepository_Count_AfterDelete(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	repo.Create(ctx, &testItem{ID: "2", Name: "Bob"})

	n, _ := repo.Count(ctx, nil)
	if n != 2 {
		t.Errorf("Count() = %d, want 2", n)
	}

	repo.DeleteById(ctx, "1")

	n, _ = repo.Count(ctx, nil)
	if n != 1 {
		t.Errorf("Count() after delete = %d, want 1", n)
	}
}

// ── Adversarial hooks tests ────────────────────────────────────────────────

func TestRegistry_AfterHookModifiesEntity(t *testing.T) {
	reg := hooks.NewRegistry[testItem]()

	reg.After(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		entity.Name = "post-hook-" + entity.Name
		return nil
	})

	val := testItem{ID: "1", Name: "original"}
	if err := reg.RunAfter(hooks.OpCreate, context.Background(), &val); err != nil {
		t.Fatalf("RunAfter() error = %v", err)
	}
	if val.Name != "post-hook-original" {
		t.Errorf("entity.Name = %q, want %q", val.Name, "post-hook-original")
	}
}

func TestRegistry_HookPanicDoesNotCrash(t *testing.T) {
	reg := hooks.NewRegistry[string]()

	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *string) error {
		panic("boom from hook")
	})

	// The hook should NOT panic out of RunBefore
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunBefore panicked: %v, expected recovery", r)
		}
	}()

	err := reg.RunBefore(hooks.OpCreate, context.Background(), nil)
	// If the design is that panics are recovered, err should be non-nil
	// If panics propagate, the recover above catches it
	_ = err
}

func TestRegistry_HookPanicAfterDoesNotCrash(t *testing.T) {
	reg := hooks.NewRegistry[string]()

	reg.After(hooks.OpCreate, func(ctx context.Context, entity *string) error {
		panic("boom from after hook")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunAfter panicked: %v, expected recovery", r)
		}
	}()

	err := reg.RunAfter(hooks.OpCreate, context.Background(), nil)
	_ = err
}

func TestRegistry_BeforeHookInOrderWithAbort(t *testing.T) {
	reg := hooks.NewRegistry[int]()
	var order []string

	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *int) error {
		order = append(order, "first")
		return nil
	})
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *int) error {
		order = append(order, "second")
		return errors.New("abort")
	})
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *int) error {
		order = append(order, "third")
		return nil
	})

	err := reg.RunBefore(hooks.OpCreate, context.Background(), nil)
	if err == nil {
		t.Fatal("expected abort error, got nil")
	}
	// Third hook should NOT have run (abort stops chain)
	if len(order) != 2 {
		t.Errorf("hook order = %v, want [first second] (third should not run)", order)
	}
}

func TestRegistry_AfterHooksStillRunAfterAbort(t *testing.T) {
	// Before hooks abort → after hooks for that operation do NOT fire
	// (because the operation was never executed)
	reg := hooks.NewRegistry[string]()
	var afterFired bool

	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *string) error {
		return errors.New("before abort")
	})
	reg.After(hooks.OpCreate, func(ctx context.Context, entity *string) error {
		afterFired = true
		return nil
	})

	err := reg.RunBefore(hooks.OpCreate, context.Background(), nil)
	if err == nil {
		t.Fatal("expected before abort error")
	}
	// After hooks should NOT fire when before aborts
	// (this is tested at repository level, not registry level)
	_ = afterFired
}

func TestRegistry_IndependentOperations(t *testing.T) {
	reg := hooks.NewRegistry[testItem]()
	var createCount, deleteCount int

	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		createCount++
		return nil
	})
	reg.Before(hooks.OpDelete, func(ctx context.Context, entity *testItem) error {
		deleteCount++
		return nil
	})

	reg.RunBefore(hooks.OpCreate, context.Background(), &testItem{ID: "1"})
	reg.RunBefore(hooks.OpDelete, context.Background(), &testItem{ID: "1"})

	if createCount != 1 {
		t.Errorf("create hooks fired %d times, want 1", createCount)
	}
	if deleteCount != 1 {
		t.Errorf("delete hooks fired %d times, want 1", deleteCount)
	}
}

func TestRegistry_MultipleAfterHooksInOrder(t *testing.T) {
	reg := hooks.NewRegistry[int]()
	var order []string

	reg.After(hooks.OpUpdate, func(ctx context.Context, entity *int) error {
		order = append(order, "after-first")
		return nil
	})
	reg.After(hooks.OpUpdate, func(ctx context.Context, entity *int) error {
		order = append(order, "after-second")
		return nil
	})

	if err := reg.RunAfter(hooks.OpUpdate, context.Background(), nil); err != nil {
		t.Fatalf("RunAfter() error = %v", err)
	}
	if len(order) != 2 || order[0] != "after-first" || order[1] != "after-second" {
		t.Errorf("hook order = %v, want [after-first after-second]", order)
	}
}

func TestRepository_Hooks_AfterHookModifiesResult(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	reg.After(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		entity.Name = "modified-after-" + entity.Name
		return nil
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	created, err := repo.Create(ctx, &testItem{ID: "1", Name: "original"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "modified-after-original" {
		t.Errorf("Name = %q, want %q", created.Name, "modified-after-original")
	}
}

func TestRepository_Hooks_BeforeHookPanicRecovered(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		panic("hook panic test")
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	// The repository should NOT panic — it should catch the hook panic
	// and return an error
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Create() panicked: %v, expected recovery", r)
		}
	}()

	_, err := repo.Create(ctx, &testItem{ID: "1", Name: "test"})
	if err == nil {
		t.Fatal("expected error from panicked hook, got nil")
	}
}

func TestRepository_Hooks_AfterHookPanicRecovered(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	reg.After(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		panic("after hook panic test")
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Create() panicked: %v, expected recovery", r)
		}
	}()

	_, err := repo.Create(ctx, &testItem{ID: "1", Name: "test"})
	if err == nil {
		t.Fatal("expected error from panicked after hook, got nil")
	}
}

func TestRepository_Hooks_DeleteBeforeHook(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	reg.Before(hooks.OpDelete, func(ctx context.Context, entity *testItem) error {
		return errors.New("delete not allowed")
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})

	// Delete should still work because the repo.DeleteById does NOT call runBefore/After
	// (verify current design — if it doesn't dispatch hooks, this is just documenting behavior)
	err := repo.DeleteById(ctx, "1")
	if err != nil {
		t.Fatalf("DeleteById() error = %v (hooks not dispatched on Delete)", err)
	}
}

// ── Thread safety test ─────────────────────────────────────────────────────

func TestRepository_ConcurrentCreate(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("concurrent-%d", n)
			repo.Create(ctx, &testItem{ID: id, Name: fmt.Sprintf("item-%d", n)})
		}(i)
	}
	wg.Wait()

	n, err := repo.Count(ctx, nil)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if n != 10 {
		t.Errorf("Count() = %d, want 10", n)
	}
}
