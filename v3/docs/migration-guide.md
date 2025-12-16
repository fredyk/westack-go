# Migration Guide: v2 → v3

**Version**: v3.0.0-alpha  
**Date**: 16 December 2025  
**Status**: In Development

## Overview

westack-go v3 introduces significant improvements to type safety and API cleanliness by using the `Instance` interface more extensively throughout the codebase. This reduces unsafe type assertions and provides a more idiomatic Go experience.

**Key Benefits**:
- ✅ 92-97% reduction in `*StatefulInstance` castings
- ✅ Improved type safety
- ✅ Cleaner API surface
- ✅ Better IDE support and autocomplete
- ✅ More idiom human: Protocol Buffer-style interface usage

## Breaking Changes Summary

### 1. EventContext.Instance Type Change

**v2**:
```go
type EventContext struct {
    Instance *StatefulInstance  // Concrete type
    // ...
}
```

**v3**:
```go
type EventContext struct {
    Instance Instance  // Interface type
    // ...
}
```

**Impact**: Code accessing `.data` or `.Model` directly on `eventContext.Instance` will break.

**Migration**:
```go
// v2 (BREAKS in v3)
inst := eventContext.Instance
rawData := inst.data  // ❌ Error: data not accessible

// v3 (CORRECT)
inst := eventContext.Instance
rawData := inst.ToJSON()  // ✅ Use public method
```

### 2. HideProperties() Added to Instance Interface

**v2**: `HideProperties()` was only on `*StatefulInstance`

**v3**: `HideProperties()` is now part of the `Instance` interface

**Impact**: Minimal - mostly internal. Custom implementations of `Instance` must add this method.

**Migration**:
```go
// v2
inst.(*StatefulInstance).HideProperties()  // Required cast

// v3
inst.HideProperties()  // Direct call
```

### 3. Build() Returns Instance

**v2**:
```go
func (m *StatefulModel) Build(...) (*StatefulInstance, error)
```

**v3**:
```go
func (m *StatefulModel) Build(...) (Instance, error)
```

**Impact**: Code casting Build() result to `*StatefulInstance` may need adjustment.

**Migration**:
```go
// v2
inst, err := model.Build(data, ctx)
concrete := inst.(*StatefulInstance)  // Often unnecessary

// v3
inst, err := model.Build(data, ctx)
// Use inst directly with Interface methods
id := inst.GetID()
json := inst.ToJSON()
```

### 4. Type Switches Simplified

**v2**: Accepted both `StatefulInstance` and `*StatefulInstance`

**v3**: Prefers `Instance` interface

**Impact**: Code passing `StatefulInstance` value (not pointer) needs to adapt.

**Migration**:
```go
// v2
model.Create(myStatefulInstance, ctx)  // Value
model.Create(&myStatefulInstance, ctx)  // Pointer

// v3  
var inst Instance = &myStatefulInstance
model.Create(inst, ctx)  // Always use interface
```

## Module Path Change

**IMPORTANT**: v3 uses a different module path per Go conventions.

**v2**:
```go
import "github.com/fredyk/westack-go/v2/model"
```

**v3**:
```go
import "github.com/fredyk/westack-go/v3/model"
```

**Migration Steps**:
1. Update all imports in your project: `v2` → `v3`
2. Run `go mod tidy` to update dependencies
3. Fix any compilation errors (type assertions, direct field access)

## Common Migration Patterns

### Pattern 1: Accessing Instance Data

```go
// v2 ❌
func handleHook(ctx *model.EventContext) error {
    data := ctx.Instance.data
    return nil
}

// v3 ✅
func handleHook(ctx *model.EventContext) error {
    data := ctx.Instance.ToJSON()
    return nil
}
```

### Pattern 2: Calling HideProperties

```go
// v2 ❌
func processInstance(inst model.Instance) {
    inst.(*model.StatefulInstance).HideProperties()
}

// v3 ✅
func processInstance(inst model.Instance) {
    inst.HideProperties()
}
```

### Pattern 3: Working with Build Results

```go
// v2 ❌
inst, _ := model.Build(data, ctx)
concrete := inst.(*model.StatefulInstance)
concrete.data["foo"] = "bar"

// v3 ✅
inst, _ := model.Build(data, ctx)
updated := inst.ToJSON()
updated["foo"] = "bar"
inst.UpdateAttributes(updated, ctx)
```

## Testing Your Migration

After migrating, run:

```bash
# Update dependencies
go mod tidy

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Check coverage
go test -cover ./...
```

## v2 Support Timeline

- **Q1-Q2 2026**: Full maintenance and bug fixes
- **Q3-Q4 2026**: Security patches only  
- **2027+**: Long Term Support (LTS) - critical security only
- **Deadline**: December 2027 (end of LTS)

## Need Help?

- **Issues**: [GitHub Issues](https://github.com/fredyk/westack-go/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fredyk/westack-go/discussions)
- **Documentation**: [v3 Docs](https://github.com/fredyk/westack-go/tree/v3/docs)

## Rollback Plan

If you encounter issues with v3, you can always rollback to v2:

```bash
# Revert imports
find . -name "*.go" -exec sed -i 's|westack-go/v3|westack-go/v2|g' {} \;

# Update go.mod
go mod edit -require=github.com/fredyk/westack-go/v2@latest
go mod tidy
```

---

**Status**: This guide will be updated as v3 development progresses.  
**Last Updated**: 16 December 2025
