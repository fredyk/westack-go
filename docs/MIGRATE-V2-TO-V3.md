# Migrating from v2 to v3

**Status**: v3.0.0-alpha in development

## Summary of Breaking Changes

v3 focuses on **type safety** and **mockability** by using `Instance` and `Model` interfaces throughout, eliminating unsafe type casts. This results in cleaner, more testable code but requires migration of existing code.

**Total changes**: 28 improvements
- *StatefulInstance casts eliminated: 12
- Type switch simplifications: 4  
- Method signatures updated: 3
- **EventContext fields using interfaces**: 2 (Instance, Model)
- **Model interface methods added**: 2 (Build, QueueOperation)

**Key benefit**: **100% mockable hooks** for testing

---

## Breaking Changes

### 1. EventContext uses interfaces for full mockability

Both `EventContext.Instance` and `EventContext.Model` are now interfaces instead of concrete types.

**v2:**
```go
type EventContext struct {
    Instance *StatefulInstance  // ❌ Concrete type
    Model    *StatefulModel     // ❌ Concrete type
}
```

**v3:**
```go
type EventContext struct {
    Instance Instance  // ✅ Interface - fully mockable
    Model    Model     // ✅ Interface - fully mockable
}
```

#### 1a. EventContext.Instance is Instance interface

`EventContext.Instance` changed from `*StatefulInstance` to `Instance` interface.

**v2 code that breaks:**
```go
func myHook(ctx *model.EventContext) error {
    modelName := ctx.Instance.Model.Name  // ❌ BREAKS
    return nil
}
```

**v3 fix:**
```go
func myHook(ctx *model.EventContext) error {
    modelName := ctx.Instance.GetModel().Name  // ✅ WORKS
    return nil
}
```

---

### 2. GetOne() returns Instance interface

**v2 signature:**
```go
func (inst *StatefulInstance) GetOne(relationName string) *StatefulInstance
```

**v3 signature:**
```go
func (inst *StatefulInstance) GetOne(relationName string) Instance
```

**Migration:**
```go
// v2
relatedUser := post.GetOne("author")
userName := relatedUser.Get("name").(string)  // ✅ Still works

// v3 - no changes needed if you only use interface methods
relatedUser := post.GetOne("author")
userName := relatedUser.Get("name").(string)  // ✅ Still works

// v2 - If you cast to *StatefulInstance
relatedUser := post.GetOne("author").(*StatefulInstance)  // ❌ BREAKS in v3
relatedUser.data["name"] = "new"  // Accessing private field

// v3 - Use interface methods instead
relatedUser := post.GetOne("author")
relatedUser.Set("name", "new")  // ✅ Use public methods
```

---

### 3. UpdateAttributes() type switch simplified

**v2 before_save hook return types:**
```go
func beforeSave(ctx *EventContext) error {
    if needsChange {
        // Could return *StatefulInstance, StatefulInstance, or Instance
        newInst := &StatefulInstance{...}
        ctx.Result = newInst  // ✅ Worked
        return nil
    }
}
```

**v3 - Only Instance or wst.M accepted:**
```go
func beforeSave(ctx *EventContext) error {
    if needsChange {
        // Only Instance or wst.M supported
        newInst := &StatefulInstance{...}
        ctx.Result = Instance(newInst)  // ✅ Works - StatefulInstance implements Instance
        // OR
        ctx.Result = wst.M{"field": "value"}  // ✅ Works
        return nil
    }
}
```

---

### 4. Create() and UpdateById() type switches simplified

Same as `UpdateAttributes()` - `before_save` hooks must return `Instance` or `wst.M`.

**v2 code:**
```go
func beforeSave(ctx *EventContext) error {
    v := ctx.Result.(StatefulInstance)  // ❌ BREAKS in v3
    return nil
}
```

**v3 fix:**
```go
func beforeSave(ctx *EventContext) error {
    v := ctx.Result.(Instance)  // ✅ WORKS
    return nil
}
```

---

### 5. HideProperties() no longer needs cast

**v2:**
```go
inst.(*StatefulInstance).HideProperties()  // ❌ Unnecessary in v3
```

**v3:**
```go
inst.HideProperties()  // ✅ Direct call - HideProperties() in Instance interface
```

---

### 6. Relations ToJSON() simplified

**v2 internal code:**
```go
relatedInstance := rawRelatedData.(*StatefulInstance).ToJSON()
```

**v3 - Uses interface:**
```go
relatedInstance := rawRelatedData.(Instance).ToJSON()
```

**No user code changes needed** - this is internal.

---

### 7. EventContext.Model is Model interface (for mockability)

**v2:**
```go
func myHook(ctx *EventContext) error {
    modelConfig := ctx.Model.Config  // ❌ BREAKS - direct field access
    return nil
}
```

**v3 fix:**
```go
func myHook(ctx *EventContext) error {
    modelConfig := ctx.Model.GetConfig()  // ✅ WORKS - use interface method
    return nil
}
```

**Available Model interface methods**:
- `GetConfig() *Config`
- `GetName() string`
- `Build(data wst.M, ctx *EventContext) (Instance, error)`
- `QueueOperation(op string, ctx *EventContext, fn func(*EventContext) error)`
- `FindMany()`, `FindById()`, `Create()`, `UpdateById()`, `DeleteById()`, `Count()`

**Mockability benefit**:
```go
// v3 - Easy mocking for tests
type MockModel struct {
    GetNameFunc func() string
}

func (m *MockModel) GetName() string { return m.GetNameFunc() }
// ... implement other interface methods

func TestMyHook(t *testing.T) {
    mock := &MockModel{
        GetNameFunc: func() string { return "TestModel" },
    }
    ctx := &EventContext{Model: mock}  // ✅ Fully mockable!
    // test hook...
}
```

---

## How to Migrate

### Step 1: Update imports
```bash
find . -name "*.go" -exec sed -i 's|westack-go/v2|westack-go/v3|g' {} \;
go get github.com/fredyk/westack-go/v3@latest
go mod tidy
```

### Step 2: Fix EventContext field access

Search for patterns:
```bash
grep -r "\.Instance\.Model\." .
grep -r "\.Instance\.data" .
grep -r "ctx\.Model\." .
```

Replace with interface methods:
- `ctx.Instance.Model` → `ctx.Instance.GetModel()`
- `ctx.Model.Config` → `ctx.Model.GetConfig()`
- `ctx.Model.Name` → `ctx.Model.GetName()`
- `inst.data[key]` → `inst.Get(key)`
- `inst.data[key] = val` → `inst.Set(key, val)`

### Step 3: Fix GetOne() casts

Search for:
```bash
grep -r "GetOne.*\.\(\*StatefulInstance\)" .
```

Remove casts:
```go
// Before
user := post.GetOne("author").(*StatefulInstance)

// After
user := post.GetOne("author")  // Already Instance interface
```

### Step 4: Fix hook return types

Search for hooks:
```bash
grep -r "ctx.Result.*StatefulInstance" .
```

Ensure only `Instance` or `wst.M`:
```go
// Before
ctx.Result = &StatefulInstance{...}  // Worked

// After  
var inst Instance = &StatefulInstance{...}
ctx.Result = inst  // Or just: ctx.Result = &StatefulInstance{...}
```

### Step 5: Test
```bash
go build ./...
go test ./...
```

---

## Benefits of v3

✅ **100% Mockable**: EventContext fully mockable for testing hooks  
✅ **Type Safety**: Fewer unsafe type assertions  
✅ **Cleaner Code**: No unnecessary casts  
✅ **Better Interfaces**: Clear contract between components  
✅ **Testability**: Easy to write unit tests with mocks  
✅ **Maintainability**: Easier to extend and test  
✅ **Performance**: Reduced runtime type checks  

---

## Compatibility

**v2 Support**: Maintained until December 2027

**Recommended**: Migrate to v3 for new projects
