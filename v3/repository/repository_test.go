package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fredyk/westack-go/v3/datasource"
	"github.com/fredyk/westack-go/v3/hooks"
)

// ── Model types for reflection edge-case tests ────────────────────────────────

type itemWithPointer struct {
	ID    string  `json:"id"`
	Name  *string `json:"name"`
	Age   int     `json:"age"`
	Bytes []byte  `json:"bytes"`
}

type itemWithInt64 struct {
	ID    string  `json:"id"`
	Seq   int64   `json:"seq"`
	Score float64 `json:"score"`
}

type itemWithTaggedUnexported struct {
	ID    string `json:"id"`
	inner innerForMap  // unexported type — field is also unexported
}

type innerForMap struct {
	Name string `json:"name"`
}

type myFloat64 float64

type myByteSlice []byte

type itemWithAliasedSlice struct {
	ID    string        `json:"id"`
	Bytes myByteSlice   `json:"bytes"`
}

type itemWithAliasedTypes struct {
	ID     string     `json:"id"`
	Score  myFloat64  `json:"score"`
	Active bool       `json:"active"`
	Amount myFloat64  `json:"amount"`
}

type nonStructInput int

// ── structToMap / mapToStruct / setField edge-case tests ─────────────────────

// ── fakeConnector — in-memory PersistedConnector for unit tests ──────────────

type fakeConnector struct {
	mu       sync.RWMutex
	store    map[string][]map[string]interface{}
	indexes  map[string]map[string]int
	pkField  string
	migrated bool

	// Error injection fields
	errCreate    error
	errFindById  error
	errFindMany  error
	errUpdateById error
	errDeleteById error
	errCount     error
	errDecode    error
	errCursorErr error
}

func (f *fakeConnector) withErrCreate(e error) *fakeConnector {
	f.errCreate = e
	return f
}
func (f *fakeConnector) withErrFindById(e error) *fakeConnector {
	f.errFindById = e
	return f
}
func (f *fakeConnector) withErrFindMany(e error) *fakeConnector {
	f.errFindMany = e
	return f
}
func (f *fakeConnector) withErrUpdateById(e error) *fakeConnector {
	f.errUpdateById = e
	return f
}
func (f *fakeConnector) withErrDeleteById(e error) *fakeConnector {
	f.errDeleteById = e
	return f
}
func (f *fakeConnector) withErrCount(e error) *fakeConnector {
	f.errCount = e
	return f
}
func (f *fakeConnector) withErrDecode(e error) *fakeConnector {
	f.errDecode = e
	return f
}
func (f *fakeConnector) withErrCursorErr(e error) *fakeConnector {
	f.errCursorErr = e
	return f
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
	if f.errCreate != nil {
		return nil, f.errCreate
	}
	f.ensureCollection(collection)
	f.mu.Lock()
	defer f.mu.Unlock()

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
	if f.errFindById != nil {
		return nil, f.errFindById
	}
	f.ensureCollection(collection)
	f.mu.RLock()
	defer f.mu.RUnlock()

	pos, ok := f.indexes[collection][fmtSprint(id)]
	if !ok {
		return nil, nil
	}
	return f.store[collection][pos], nil
}

func (f *fakeConnector) FindMany(_ context.Context, collection string, query *datasource.Query) (datasource.Cursor, error) {
	if f.errFindMany != nil {
		return nil, f.errFindMany
	}
	f.ensureCollection(collection)
	f.mu.RLock()
	defer f.mu.RUnlock()

	var results []map[string]interface{}
	for _, doc := range f.store[collection] {
		results = append(results, doc)
	}
	return &fakeCursor{results: results, decodeErr: f.errDecode, cursorErr: f.errCursorErr}, nil
}

func (f *fakeConnector) Count(_ context.Context, collection string, filter *datasource.Filter) (int64, error) {
	if f.errCount != nil {
		return 0, f.errCount
	}
	f.ensureCollection(collection)
	f.mu.RLock()
	defer f.mu.RUnlock()
	return int64(len(f.store[collection])), nil
}

func (f *fakeConnector) UpdateById(_ context.Context, collection string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	if f.errUpdateById != nil {
		return nil, f.errUpdateById
	}
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
	if f.errDeleteById != nil {
		return 0, f.errDeleteById
	}
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
	results   []map[string]interface{}
	pos       int
	decodeErr error
	cursorErr error
}

func (c *fakeCursor) Next(_ context.Context) bool {
	c.pos++
	return c.pos <= len(c.results)
}

func (c *fakeCursor) Decode(val interface{}) error {
	if c.decodeErr != nil {
		return c.decodeErr
	}
	if m, ok := val.(*map[string]interface{}); ok {
		*m = c.results[c.pos-1]
	}
	return nil
}

func (c *fakeCursor) All(_ context.Context, val interface{}) error { return nil }
func (c *fakeCursor) Close(_ context.Context) error                { return nil }
func (c *fakeCursor) Err() error                                   { return c.cursorErr }

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
	NoTag     string    // exported, no json tag → key="", second continue path
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

// ── structToMap / mapToStruct / setField edge-case tests ─────────────────────

func TestStructToMap_NilPointer(t *testing.T) {
	var p *testItem = nil
	m, err := structToMap(p)
	if err != nil {
		t.Fatalf("structToMap(nil pointer) error = %v, want nil", err)
	}
	if m != nil {
		t.Errorf("structToMap(nil pointer) = %v, want nil", m)
	}
}

func TestStructToMap_NonStruct(t *testing.T) {
	_, err := structToMap(42)
	if err == nil {
		t.Fatal("structToMap(int) returned nil error, want error")
	}
	want := "expected struct"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("structToMap(int) error = %v, want error containing %q", err, want)
	}
}

func TestStructToMap_UnexportedFieldWithJsonTag_CanInterfaceFalse(t *testing.T) {
	// Unexported field with json tag: key non-empty, CanInterface()=false
	// → structToMap should skip it (hit CanInterface false path)
	result, err := structToMap(itemWithTaggedUnexported{ID: "1"})
	if err != nil {
		t.Fatalf("structToMap error = %v", err)
	}
	if _, ok := result["tagged_unexported"]; ok {
		t.Error("unexported field with json tag should not appear in map (CanInterface=false)")
	}
	if result["id"] != "1" {
		t.Errorf("id = %v, want %q", result["id"], "1")
	}
}

func TestStructToMap_ExportedFieldNoJsonTag_Skipped(t *testing.T) {
	// Exported field with no json tag: key == "", IsExported == true → second continue
	result, err := structToMap(testItem{ID: "1", NoTag: "should-be-skipped"})
	if err != nil {
		t.Fatalf("structToMap error = %v", err)
	}
	if _, ok := result["NoTag"]; ok {
		t.Error("exported field with no json tag should not appear in map")
	}
	if result["id"] != "1" {
		t.Errorf("id = %v, want %q", result["id"], "1")
	}
}

func TestMapToStruct_UnexportedFields_Skipped(t *testing.T) {
	// testItem has unexported field Internal (no json tag) — mapToStruct should skip it
	doc := map[string]interface{}{
		"id":       "1",
		"name":     "Test",
		"Internal": "should-be-skipped",
	}
	result := testItem{}
	err := mapToStruct(doc, &result)
	if err != nil {
		t.Fatalf("mapToStruct error = %v", err)
	}
	if result.ID != "1" {
		t.Errorf("ID = %q, want %q", result.ID, "1")
	}
	if result.Name != "Test" {
		t.Errorf("Name = %q, want %q", result.Name, "Test")
	}
}

func TestMapToStruct_UnexportedFieldWithJsonTag_CanSetFalse(t *testing.T) {
	// Unexported field with json tag: key is non-empty, IsExported=false, CanSet=false
	// → mapToStruct should hit the CanSet() false path and skip
	doc := map[string]interface{}{
		"id":                  "1",
		"tagged_unexported":   "should-be-skipped",
	}
	result := itemWithTaggedUnexported{}
	err := mapToStruct(doc, &result)
	if err != nil {
		t.Fatalf("mapToStruct error = %v", err)
	}
	if result.ID != "1" {
		t.Errorf("ID = %q, want %q", result.ID, "1")
	}
}

func TestSetField_PointerField(t *testing.T) {
	name := "hello"
	item := itemWithPointer{ID: "1", Name: nil}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Name")

	err := setField(field, name)
	if err != nil {
		t.Fatalf("setField on *string field error = %v", err)
	}
	if item.Name == nil {
		t.Fatal("Name pointer should have been allocated")
	}
	if *item.Name != "hello" {
		t.Errorf("*Name = %q, want %q", *item.Name, "hello")
	}
}

func TestSetField_Float64ToInt(t *testing.T) {
	item := itemWithPointer{ID: "1", Age: 0}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Age")

	err := setField(field, float64(42.7))
	if err != nil {
		t.Fatalf("setField float64→int error = %v", err)
	}
	if item.Age != 42 {
		t.Errorf("Age = %d, want 42", item.Age)
	}
}

func TestSetField_ByteSlice(t *testing.T) {
	item := itemWithPointer{ID: "1", Bytes: nil}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Bytes")

	data := []byte("hello-bytes")
	err := setField(field, data)
	if err != nil {
		t.Fatalf("setField []byte error = %v", err)
	}
	if len(item.Bytes) != 11 {
		t.Errorf("len(Bytes) = %d, want 11", len(item.Bytes))
	}
}

func TestSetField_TypeMismatch_SilentSkip(t *testing.T) {
	// int field receiving a string value — setField should silently skip (no error)
	item := itemWithPointer{ID: "1", Age: 0}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Age")

	err := setField(field, "not-a-number")
	if err != nil {
		t.Fatalf("setField type mismatch should return nil error, got = %v", err)
	}
	if item.Age != 0 {
		t.Errorf("Age = %d, want 0 (should be silently skipped)", item.Age)
	}
}

func TestSetField_Int64FieldFromInt(t *testing.T) {
	item := itemWithInt64{ID: "1", Seq: 0}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Seq")

	// raw is int (not int64), so exact-type match fails and switch case fires
	err := setField(field, int(99))
	if err != nil {
		t.Fatalf("setField int64 field from int error = %v", err)
	}
	if item.Seq != 99 {
		t.Errorf("Seq = %d, want 99", item.Seq)
	}
}

func TestSetField_IntFieldFromInt64(t *testing.T) {
	item := itemWithPointer{ID: "1", Age: 0}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Age")

	err := setField(field, int64(77))
	if err != nil {
		t.Fatalf("setField int field from int64 error = %v", err)
	}
	if item.Age != 77 {
		t.Errorf("Age = %d, want 77", item.Age)
	}
}

func TestSetField_Float64FieldFromAliasedFloat64(t *testing.T) {
	// myFloat64 is an alias of float64. When mapToStruct reads float64 from map
	// and writes to myFloat64 field, rawVal.Type() (float64) != fieldVal.Type() (myFloat64),
	// so the switch case reflect.Float64 fires and raw.(float64) succeeds.
	item := itemWithAliasedTypes{ID: "1"}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Score")

	err := setField(field, float64(3.14))
	if err != nil {
		t.Fatalf("setField float64 field from float64 raw with aliased type error = %v", err)
	}
	if float64(item.Score) != 3.14 {
		t.Errorf("Score = %f, want 3.14", float64(item.Score))
	}
}

func TestSetField_StringField(t *testing.T) {
	item := itemWithPointer{ID: "1", Name: nil}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("ID")

	err := setField(field, "new-id")
	if err != nil {
		t.Fatalf("setField string error = %v", err)
	}
	if item.ID != "new-id" {
		t.Errorf("ID = %q, want %q", item.ID, "new-id")
	}
}

func TestSetField_BoolField(t *testing.T) {
	item := testItem{ID: "1", Active: false}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Active")

	err := setField(field, true)
	if err != nil {
		t.Fatalf("setField bool error = %v", err)
	}
	if !item.Active {
		t.Error("Active = false, want true")
	}
}

func TestSetField_SliceFieldFromByteSlice(t *testing.T) {
	// myByteSlice is an alias of []byte. rawVal.Type() ([]byte) != fieldVal.Type() (myByteSlice),
	// so the switch case reflect.Slice fires and raw.([]byte) succeeds.
	item := itemWithAliasedSlice{ID: "1"}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Bytes")

	err := setField(field, []byte("slice-data"))
	if err != nil {
		t.Fatalf("setField []byte field from []byte raw with aliased type error = %v", err)
	}
	if string(item.Bytes) != "slice-data" {
		t.Errorf("Bytes = %q, want %q", string(item.Bytes), "slice-data")
	}
}

func TestSetField_DefaultNoMatch(t *testing.T) {
	// int field receiving a struct — no case matches, should silently return nil
	type itemWithMap struct {
		ID   string `json:"id"`
		Data int    `json:"data"`
	}
	item := itemWithMap{ID: "1", Data: 0}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Data")

	err := setField(field, map[string]int{"x": 1})
	if err != nil {
		t.Fatalf("setField type mismatch should return nil error, got = %v", err)
	}
	if item.Data != 0 {
		t.Errorf("Data = %d, want 0 (should be silently skipped)", item.Data)
	}
}

func TestSetField_Float64FieldDirect(t *testing.T) {
	item := testItem{ID: "1", Score: 0}
	rv := reflect.ValueOf(&item).Elem()
	field := rv.FieldByName("Score")

	err := setField(field, float64(42.5))
	if err != nil {
		t.Fatalf("setField float64 error = %v", err)
	}
	if item.Score != 42.5 {
		t.Errorf("Score = %f, want 42.5", item.Score)
	}
}

func TestRepository_Create_Int64RoundTrip(t *testing.T) {
	conn := newFakeConnector()
	repo := New[itemWithInt64](conn, datasource.ModelDef{
		Name:       "ItemWithInt64",
		Collection: "item_with_int64s",
		Properties: []datasource.PropertyDef{
			{Name: "id", Type: datasource.PropString, PrimaryKey: true},
			{Name: "seq", Type: datasource.PropInt64},
			{Name: "score", Type: datasource.PropFloat64},
		},
	})
	ctx := context.Background()

	created, err := repo.Create(ctx, &itemWithInt64{ID: "1", Seq: 42, Score: 3.14})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Seq != 42 {
		t.Errorf("Seq = %d, want 42", created.Seq)
	}
	if created.Score != 3.14 {
		t.Errorf("Score = %f, want 3.14", created.Score)
	}
}

// ── Connector error path tests ───────────────────────────────────────────────

func TestRepository_Create_ConnectorError(t *testing.T) {
	connErr := errors.New("db connection refused")
	conn := newFakeConnector().withErrCreate(connErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	_, err := repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if err == nil {
		t.Fatal("Create() expected error, got nil")
	}
	if !errors.Is(err, connErr) {
		t.Errorf("Create() error = %v, want wrapped %v", err, connErr)
	}
}

func TestRepository_FindById_ConnectorError(t *testing.T) {
	connErr := errors.New("db timeout")
	conn := newFakeConnector().withErrFindById(connErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	_, err := repo.FindById(ctx, "1")
	if err == nil {
		t.Fatal("FindById() expected error, got nil")
	}
	if !errors.Is(err, connErr) {
		t.Errorf("FindById() error = %v, want wrapped %v", err, connErr)
	}
}

func TestRepository_FindMany_ConnectorError(t *testing.T) {
	connErr := errors.New("query failed")
	conn := newFakeConnector().withErrFindMany(connErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	_, err := repo.FindMany(ctx, nil)
	if err == nil {
		t.Fatal("FindMany() expected error, got nil")
	}
	if !errors.Is(err, connErr) {
		t.Errorf("FindMany() error = %v, want wrapped %v", err, connErr)
	}
}

func TestRepository_UpdateById_ConnectorError(t *testing.T) {
	connErr := errors.New("update failed")
	conn := newFakeConnector().withErrUpdateById(connErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	_, err := repo.UpdateById(ctx, "1", &testItem{Name: "Bob"})
	if err == nil {
		t.Fatal("UpdateById() expected error, got nil")
	}
	if !errors.Is(err, connErr) {
		t.Errorf("UpdateById() error = %v, want wrapped %v", err, connErr)
	}
}

func TestRepository_DeleteById_ConnectorError(t *testing.T) {
	connErr := errors.New("delete failed")
	conn := newFakeConnector().withErrDeleteById(connErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	err := repo.DeleteById(ctx, "1")
	if err == nil {
		t.Fatal("DeleteById() expected error, got nil")
	}
	if !errors.Is(err, connErr) {
		t.Errorf("DeleteById() error = %v, want wrapped %v", err, connErr)
	}
}

func TestRepository_Count_ConnectorError(t *testing.T) {
	connErr := errors.New("count failed")
	conn := newFakeConnector().withErrCount(connErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	_, err := repo.Count(ctx, nil)
	if err == nil {
		t.Fatal("Count() expected error, got nil")
	}
	if !errors.Is(err, connErr) {
		t.Errorf("Count() error = %v, want wrapped %v", err, connErr)
	}
}

// ── Hook error path tests ────────────────────────────────────────────────────

func TestRepository_Hooks_AfterCreateErrors(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	afterErr := errors.New("after hook failed")
	reg.After(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		return afterErr
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	_, err := repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if err == nil {
		t.Fatal("Create() expected error from after hook, got nil")
	}
	if !errors.Is(err, afterErr) {
		t.Errorf("Create() error = %v, want wrapped %v", err, afterErr)
	}
}

func TestRepository_Hooks_AfterUpdateErrors(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	afterErr := errors.New("after update hook failed")
	reg.After(hooks.OpUpdate, func(ctx context.Context, entity *testItem) error {
		return afterErr
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	_, err := repo.UpdateById(ctx, "1", &testItem{Name: "Bob"})
	if err == nil {
		t.Fatal("UpdateById() expected error from after hook, got nil")
	}
	if !errors.Is(err, afterErr) {
		t.Errorf("UpdateById() error = %v, want wrapped %v", err, afterErr)
	}
}

func TestRepository_Hooks_MultipleHooks_SecondFails(t *testing.T) {
	conn := newFakeConnector()
	reg := hooks.NewRegistry[testItem]()
	var firstRan bool
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		firstRan = true
		return nil
	})
	secondErr := errors.New("second hook failed")
	reg.Before(hooks.OpCreate, func(ctx context.Context, entity *testItem) error {
		return secondErr
	})
	repo := NewWithHooks[testItem](conn, testModelDef, reg)
	ctx := context.Background()

	_, err := repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})
	if err == nil {
		t.Fatal("Create() expected error from second hook, got nil")
	}
	if !errors.Is(err, secondErr) {
		t.Errorf("Create() error = %v, want wrapped %v", err, secondErr)
	}
	if !firstRan {
		t.Error("first hook should have run before the second failed")
	}
}

// ── Edge cases ───────────────────────────────────────────────────────────────

func TestRepository_UpdateById_ReturnsNilForMissingID(t *testing.T) {
	conn := newFakeConnector()
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	updated, err := repo.UpdateById(ctx, "nonexistent", &testItem{Name: "Bob"})
	if err != nil {
		t.Fatalf("UpdateById() error = %v, want nil", err)
	}
	if updated != nil {
		t.Errorf("UpdateById() on missing ID = %v, want nil", updated)
	}
}

func TestRepository_FindMany_CursorErrReturnsError(t *testing.T) {
	cursorErr := errors.New("cursor iteration error")
	conn := newFakeConnector().withErrCursorErr(cursorErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	// Create an item first so there's something to iterate
	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})

	_, err := repo.FindMany(ctx, nil)
	if err == nil {
		t.Fatal("FindMany() expected cursor.Err() error, got nil")
	}
	if !errors.Is(err, cursorErr) {
		t.Errorf("FindMany() error = %v, want wrapped %v", err, cursorErr)
	}
}

func TestRepository_FindMany_CursorDecodeError(t *testing.T) {
	decodeErr := errors.New("decode error")
	conn := newFakeConnector().withErrDecode(decodeErr)
	repo := New[testItem](conn, testModelDef)
	ctx := context.Background()

	repo.Create(ctx, &testItem{ID: "1", Name: "Alice"})

	_, err := repo.FindMany(ctx, nil)
	if err == nil {
		t.Fatal("FindMany() expected cursor.Decode() error, got nil")
	}
	if !errors.Is(err, decodeErr) {
		t.Errorf("FindMany() error = %v, want wrapped %v", err, decodeErr)
	}
}

func TestRepository_UpdateById_StructToMapError(t *testing.T) {
	conn := newFakeConnector()
	ctx := context.Background()

	type intAlias int
	repoInt := New[intAlias](conn, datasource.ModelDef{
		Name:       "IntAlias",
		Collection: "int_aliases",
		Properties: []datasource.PropertyDef{
			{Name: "id", Type: datasource.PropString, PrimaryKey: true},
		},
	})

	_, err := repoInt.UpdateById(ctx, "1", new(intAlias))
	if err == nil {
		t.Fatal("UpdateById() expected error for non-struct type, got nil")
	}
	if !strings.Contains(err.Error(), "structToMap") {
		t.Errorf("UpdateById() error = %v, want wrapped structToMap error", err)
	}
}

func TestRepository_Create_StructToMapError(t *testing.T) {
	conn := newFakeConnector()
	ctx := context.Background()

	// Use a type alias of int — when Create calls structToMap on *intAlias,
	// rv.Kind() is Ptr → Elem() → Kind() is Int (not Struct) → error.
	type intAlias int
	repoInt := New[intAlias](conn, datasource.ModelDef{
		Name:       "IntAlias",
		Collection: "int_aliases",
		Properties: []datasource.PropertyDef{
			{Name: "id", Type: datasource.PropString, PrimaryKey: true},
		},
	})

	_, err := repoInt.Create(ctx, new(intAlias))
	if err == nil {
		t.Fatal("Create() expected error for non-struct type, got nil")
	}
	if !strings.Contains(err.Error(), "structToMap") {
		t.Errorf("Create() error = %v, want wrapped structToMap error", err)
	}
}

// ── setField conversion coverage: store float64/int64 in fakeConnector
// so that mapToStruct → setField must convert types ───────────────────────────

type conversionTarget struct {
	ID    string  `json:"id"`
	Age   int     `json:"age"`
	Seq   int64   `json:"seq"`
}

var conversionModelDef = datasource.ModelDef{
	Name:       "ConversionTarget",
	Collection: "conversion_targets",
	Properties: []datasource.PropertyDef{
		{Name: "id", Type: datasource.PropString, PrimaryKey: true},
		{Name: "age", Type: datasource.PropInt},
		{Name: "seq", Type: datasource.PropInt64},
	},
}

func TestSetField_ConversionsViaFindById(t *testing.T) {
	conn := newFakeConnector()
	repo := New[conversionTarget](conn, conversionModelDef)
	ctx := context.Background()

	// Pre-populate the store with values of DIFFERENT types than the struct fields.
	// This forces mapToStruct → setField to do type conversions.
	conn.ensureCollection("conversion_targets")
	conn.mu.Lock()
	conn.store["conversion_targets"] = []map[string]interface{}{
		{
			"id":  "conv-1",
			"age": float64(99),  // float64 → int (conversion path via line 269)
			"seq": int(42),      // int → int64 (conversion path via line 268)
		},
	}
	conn.indexes["conversion_targets"] = map[string]int{"conv-1": 0}
	conn.mu.Unlock()

	found, err := repo.FindById(ctx, "conv-1")
	if err != nil {
		t.Fatalf("FindById() error = %v", err)
	}
	if found == nil {
		t.Fatal("FindById() returned nil, want item")
	}
	if found.Age != 99 {
		t.Errorf("Age = %d, want 99 (from float64→int conversion)", found.Age)
	}
	if found.Seq != 42 {
		t.Errorf("Seq = %d, want 42 (from int→int64 conversion)", found.Seq)
	}
}

func TestSetField_Int64FieldFromFloat64(t *testing.T) {
	type itemWithFloatToInt struct {
		ID  string `json:"id"`
		Val int64  `json:"val"`
	}
	conn := newFakeConnector()
	repo := New[itemWithFloatToInt](conn, datasource.ModelDef{
		Name:       "FloatToInt",
		Collection: "float_to_ints",
		Properties: []datasource.PropertyDef{
			{Name: "id", Type: datasource.PropString, PrimaryKey: true},
			{Name: "val", Type: datasource.PropFloat64},
		},
	})
	ctx := context.Background()

	conn.ensureCollection("float_to_ints")
	conn.mu.Lock()
	conn.store["float_to_ints"] = []map[string]interface{}{
		{"id": "1", "val": float64(42.9)}, // float64 → int64 conversion
	}
	conn.indexes["float_to_ints"] = map[string]int{"1": 0}
	conn.mu.Unlock()

	found, err := repo.FindById(ctx, "1")
	if err != nil {
		t.Fatalf("FindById() error = %v", err)
	}
	if found.Val != 42 {
		t.Errorf("Val = %d, want 42 (from float64→int64 conversion)", found.Val)
	}
}

// ── Compile-time interface check ─────────────────────────────────────────────

var _ datasource.PersistedConnector = (*fakeConnector)(nil)
