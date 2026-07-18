package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fredyk/westack-go/v3/datasource"
	"github.com/fredyk/westack-go/v3/hooks"
)

// ── fakeConnector — in-memory PersistedConnector for unit tests ──────────────

type fakeConnector struct {
	mu       sync.RWMutex
	store    map[string][]map[string]interface{}
	indexes  map[string]map[string]int
	pkField  string
	migrated bool
}

func newFakeConnector() *fakeConnector {
	return &fakeConnector{
		store:   make(map[string][]map[string]interface{}),
		indexes: make(map[string]map[string]int),
		pkField: "id",
	}
}

func (f *fakeConnector) ensureCollection(c string) {
	if _, ok := f.store[c]; !ok {
		f.store[c] = nil
		f.indexes[c] = make(map[string]int)
	}
}

func (f *fakeConnector) GetName() string                                        { return "fake" }
func (f *fakeConnector) Connect(context.Context) error                         { return nil }
func (f *fakeConnector) Disconnect() error                                     { return nil }
func (f *fakeConnector) Ping(context.Context) error                            { return nil }
func (f *fakeConnector) CreateMany(context.Context, string, []map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeConnector) DeleteMany(context.Context, string, *datasource.Filter) (int64, error) {
	return 0, nil
}
func (f *fakeConnector) Migrate(_ context.Context, _ datasource.ModelDef) error {
	f.migrated = true
	return nil
}

func (f *fakeConnector) Create(_ context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.ensureCollection(collection)

	doc := make(map[string]interface{}, len(data))
	for k, v := range data {
		doc[k] = v
	}
	f.store[collection] = append(f.store[collection], doc)
	pos := len(f.store[collection]) - 1
	id := fmtSprint(doc[f.pkField])
	f.indexes[collection][id] = pos
	return doc, nil
}

func (f *fakeConnector) FindById(_ context.Context, collection string, id interface{}) (map[string]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureCollection(collection)

	pos, ok := f.indexes[collection][fmtSprint(id)]
	if !ok {
		return nil, nil
	}
	return f.store[collection][pos], nil
}

func (f *fakeConnector) FindMany(_ context.Context, collection string, query *datasource.Query) (datasource.Cursor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureCollection(collection)

	var results []map[string]interface{}
	for _, doc := range f.store[collection] {
		results = append(results, doc)
	}
	return &fakeCursor{results: results}, nil
}

func (f *fakeConnector) Count(_ context.Context, collection string, filter *datasource.Filter) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureCollection(collection)
	return int64(len(f.store[collection])), nil
}

func (f *fakeConnector) UpdateById(_ context.Context, collection string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	f.ensureCollection(collection)
	f.mu.Lock()
	defer f.mu.Unlock()

	pos, ok := f.indexes[collection][fmtSprint(id)]
	if !ok {
		return nil, nil
	}

	doc := f.store[collection][pos]
	for k, v := range data {
		doc[k] = v
	}
	return doc, nil
}

func (f *fakeConnector) DeleteById(_ context.Context, collection string, id interface{}) (int64, error) {
	f.ensureCollection(collection)
	f.mu.Lock()
	defer f.mu.Unlock()

	pos, ok := f.indexes[collection][fmtSprint(id)]
	if !ok {
		return 0, nil
	}

	f.store[collection] = append(f.store[collection][:pos], f.store[collection][pos+1:]...)
	delete(f.indexes[collection], fmtSprint(id))
	for i, doc := range f.store[collection] {
		f.indexes[collection][fmtSprint(doc[f.pkField])] = i
	}
	return 1, nil
}

func fmtSprint(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// ── fakeCursor ───────────────────────────────────────────────────────────────

type fakeCursor struct {
	results []map[string]interface{}
	pos     int
}

func (c *fakeCursor) Next(_ context.Context) bool {
	c.pos++
	return c.pos <= len(c.results)
}

func (c *fakeCursor) Decode(val interface{}) error {
	if m, ok := val.(*map[string]interface{}); ok {
		*m = c.results[c.pos-1]
	}
	return nil
}

func (c *fakeCursor) All(_ context.Context, val interface{}) error { return nil }
func (c *fakeCursor) Close(_ context.Context) error                { return nil }
func (c *fakeCursor) Err() error                                  { return nil }

// ── Test model types ─────────────────────────────────────────────────────────

type testItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Active    bool      `json:"active"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Internal  string    // no json tag → should be skipped
	Ignored   string    `json:"-"` // explicit skip
}

var testModelDef = datasource.ModelDef{
	Name:       "TestItem",
	Collection: "test_items",
	Properties: []datasource.PropertyDef{
		{Name: "id", Type: datasource.PropString, PrimaryKey: true},
		{Name: "name", Type: datasource.PropString},
		{Name: "age", Type: datasource.PropInt},
		{Name: "active", Type: datasource.PropBool},
		{Name: "score", Type: datasource.PropFloat64},
		{Name: "created_at", Type: datasource.PropTime},
		{Name: "updated_at", Type: datasource.PropTime},
	},
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestRepository_Create_And_FindById(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	item := &testItem{
		ID:   "item-1",
		Name: "Alice",
		Age:  30,
	}

	created, err := repo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != "item-1" {
		t.Errorf("Create() ID = %q, want %q", created.ID, "item-1")
	}
	if created.Name != "Alice" {
		t.Errorf("Create() Name = %q, want %q", created.Name, "Alice")
	}

	found, err := repo.FindById(ctx, "item-1")
	if err != nil {
		t.Fatalf("FindById() error = %v", err)
	}
	if found == nil {
		t.Fatal("FindById() returned nil, want item")
	}
	if found.Name != "Alice" {
		t.Errorf("FindById() Name = %q, want %q", found.Name, "Alice")
	}
	if found.Age != 30 {
		t.Errorf("FindById() Age = %d, want %d", found.Age, 30)
	}
}

func TestRepository_FindById_NotFound(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	found, err := repo.FindById(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindById() error = %v, want nil", err)
	}
	if found != nil {
		t.Errorf("FindById() = %v, want nil", found)
	}
}

func TestRepository_FindMany(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	repo.Create(ctx, &testItem{ID: "2", Name: "Bob"})
	repo.Create(ctx, &testItem{ID: "3", Name: "Charlie"})

	items, err := repo.FindMany(ctx, nil)
	if err != nil {
		t.Fatalf("FindMany() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("FindMany() returned %d items, want 3", len(items))
	}
}

func TestRepository_UpdateById(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice", Age: 30})

	updated, err := repo.UpdateById(ctx, "1", &testItem{Name: "Alice Updated", Age: 31})
	if err != nil {
		t.Fatalf("UpdateById() error = %v", err)
	}
	if updated.Name != "Alice Updated" {
		t.Errorf("UpdateById() Name = %q, want %q", updated.Name, "Alice Updated")
	}
	if updated.Age != 31 {
		t.Errorf("UpdateById() Age = %d, want %d", updated.Age, 31)
	}

	// Verify persisted
	found, _ := repo.FindById(ctx, "1")
	if found.Name != "Alice Updated" {
		t.Errorf("FindAfterUpdate() Name = %q, want %q", found.Name, "Alice Updated")
	}
}

func TestRepository_DeleteById(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})

	err := repo.DeleteById(ctx, "1")
	if err != nil {
		t.Fatalf("DeleteById() error = %v", err)
	}

	found, _ := repo.FindById(ctx, "1")
	if found != nil {
		t.Errorf("FindAfterDelete() = %v, want nil", found)
	}
}

func TestRepository_Mapping_JsonTags(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	item := &testItem{ID: "1", Name: "Test", Internal: "secret", Ignored: "skip"}
	created, err := repo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Internal field (no json tag) should not be in the stored map
	raw := conn.store["test_items"][0]
	if _, ok := raw["Internal"]; ok {
		t.Error("unexported/no-tag field 'Internal' should not be mapped to storage")
	}
	if _, ok := raw["Ignored"]; ok {
		t.Error("json:\"-\" field 'Ignored' should not be mapped to storage")
	}
	_ = created
}

func TestRepository_Mapping_TimeRoundTrip(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	item := &testItem{ID: "1", Name: "TimeTest", CreatedAt: now, UpdatedAt: now}
	repo.Create(ctx, item)

	found, _ := repo.FindById(ctx, "1")
	if !found.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt roundtrip: got %v, want %v", found.CreatedAt, now)
	}
	if !found.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt roundtrip: got %v, want %v", found.UpdatedAt, now)
	}
}

func TestRepository_Mapping_BoolAndFloat(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	item := &testItem{ID: "1", Active: true, Score: 99.5}
	repo.Create(ctx, item)

	found, _ := repo.FindById(ctx, "1")
	if !found.Active {
		t.Error("Active roundtrip: got false, want true")
	}
	if found.Score != 99.5 {
		t.Errorf("Score roundtrip: got %f, want 99.5", found.Score)
	}
}

func TestRepository_Count(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	n, err := repo.Count(ctx, nil)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if n != 0 {
		t.Errorf("Count() on empty = %d, want 0", n)
	}

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	repo.Create(ctx, &testItem{ID: "2", Name: "Bob"})

	n, err = repo.Count(ctx, nil)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if n != 2 {
		t.Errorf("Count() = %d, want 2", n)
	}
}

// ── Hook integration tests ──────────────────────────────────────────────────

func TestRepository_Hooks_BeforeCreateFiresAndModifies(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		entity.Name = "hooked-" + entity.Name
		return nil
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	created, err := repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "hooked-Alice" {
		t.Errorf("Name = %q, want %q", created.Name, "hooked-Alice")
	}
}

func TestRepository_Hooks_BeforeCreateAborts(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	abortErr := errors.New("validation rejected")
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		return abortErr
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	_, err := repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if !errors.Is(err, abortErr) {
		t.Errorf("Create() error = %v, want %v", err, abortErr)
	}
	// Nothing should be stored
	n, _ := repo.Count(ctx, nil)
	if n != 0 {
		t.Errorf("Count() = %d, want 0 (operation should have been aborted)", n)
	}
}

func TestRepository_Hooks_AfterCreateFires(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	var afterName string
	reg.After(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		afterName = entity.Name
		return nil
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if afterName != "Alice" {
		t.Errorf("after hook saw Name = %q, want %q", afterName, "Alice")
	}
}

func TestRepository_Hooks_BeforeUpdateAborts(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	abortErr := errors.New("update denied")
	reg.Before(hooks.OpUpdate, func(ctx context.Context, entity *testItem) error {
		return abortErr
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	_, err := repo.UpdateById(ctx, "1", &testItem{Name: "Bob"})
	if !errors.Is(err, abortErr) {
		t.Errorf("UpdateById() error = %v, want %v", err, abortErr)
	}
}

func TestRepository_Hooks_NoRegistry_IsNoOp(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	created, err := repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "Alice" {
		t.Errorf("Name = %q, want %q", created.Name, "Alice")
	}
}

// ── Compile-time interface check ─────────────────────────────────────────────

var _ datasource.PersistedConnector = (*fakeConnector)(nil)
