# westack-go v3: Evolution Proposal (2025)

## Executive Summary

**Current State (v2):** westack-go es un framework REST inspirado en LoopBack (2018), con modelo-driven development, RBAC robusto, y arquitectura en capas. Diseñado para APIs tradicionales con MongoDB.

**Proposed State (v3):** Framework multi-protocolo, event-driven, AI-ready, con observabilidad nativa, manteniendo la simplicidad y productividad de v2.

**Core Value Proposition v3:**  
*"Build modern APIs in Go with enterprise features out-of-the-box: GraphQL + REST + gRPC + Real-time, from a single source of truth."*

---

## Gap Analysis: 2018 → 2025

### Current Strengths (Mantener) ✅
1. **Simplicidad LoopBack-style**: Modelo → API automática
2. **RBAC robusto**: Casbin integration, bearer token propagation
3. **Go performance**: Type-safe, compiled, eficiente
4. **Extensibilidad**: Hooks (before/after), custom endpoints
5. **Developer Experience**: CLI tools, Swagger automático

### Critical Gaps (Resolver) ⚠️

#### 1. **Protocolo Único (Solo REST)**
- **Problema**: En 2025, GraphQL es estándar para frontends modernos, gRPC para microservicios
- **Impacto**: Desarrolladores construyen wrappers GraphQL sobre REST → complejidad
- **Competidores**: Hasura (GraphQL instant), gRPC Gateway, Buf Connect

#### 2. **Sin Real-time Nativo**
- **Problema**: WebSockets/SSE son añadidos manuales, no first-class
- **Impacto**: Apps modernas requieren real-time (chats, notifications, live dashboards)
- **Competidores**: Supabase (real-time by default), Firebase

#### 3. **Datasources Limitados**
- **Problema**: Solo MongoDB + in-memory
- **Impacto**: No soporta Postgres (dominante en 2025), SQLite (edge), Vector DBs (AI)
- **Competidores**: GORM (multi-DB), Ent (type-safe para cualquier SQL)

#### 4. **Sin Observabilidad Moderna**
- **Problema**: No hay traces, metrics, logs estructurados (OpenTelemetry)
- **Impacto**: Debugging en producción es manual y lento
- **Estándar 2025**: OpenTelemetry es esperado en frameworks enterprise

#### 5. **Sin AI/LLM Patterns**
- **Problema**: No hay soporte para embeddings, vector search, streaming responses
- **Impacto**: Developers construyen RAG y AI features desde cero
- **Tendencia 2025**: APIs AI-native (streaming LLM responses, semantic search)

#### 6. **Arquitectura Monolítica**
- **Problema**: No hay patterns para event-driven, CQRS, service mesh
- **Impacto**: Difícil migrar a microservicios o arquitecturas distribuidas
- **Estándar 2025**: Event-driven como opción, no afterthought

#### 7. **JSON Models → Go Indirection**
- **Problema**: JSON como source of truth → regeneración manual, no type-safe
- **Impacto**: Refactoring es manual, errores en runtime
- **Tendencia 2025**: Code-first (Go structs) con tooling automático

---

## v3 Architecture: "Ports & Adapters" Moderno

### Principio Fundamental
**"Define once, expose everywhere"** - Un modelo → múltiples protocolos automáticamente

### Capas Propuestas

```
┌─────────────────────────────────────────────────────────────┐
│                    PROTOCOL ADAPTERS                        │
│  REST (Fiber/Chi) │ GraphQL (gqlgen) │ gRPC (Buf/protoc)  │
│         WebSockets (gorilla)  │  SSE (native)               │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                      DOMAIN LAYER                           │
│  Models (Go structs) │ Hooks │ Business Logic │ RBAC       │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                    EVENT BUS (Optional)                     │
│      NATS │ Kafka │ Redis Streams │ In-Memory              │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                   DATASOURCE PLUGINS                        │
│  MongoDB │ Postgres │ SQLite │ Vector (Pinecone/Weaviate) │
│           Redis │ S3 │ Custom                              │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                   OBSERVABILITY LAYER                       │
│    OpenTelemetry (Traces/Metrics/Logs) │ Health Checks     │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Features v3

### 1. **Multi-Protocol Exposure** 🌐

**Ejemplo:**
```go
// Define model once (pure Go)
type Note struct {
    westack.Model
    ID      string    `json:"id" gql:"id" grpc:"id"`
    Title   string    `json:"title" gql:"title!" grpc:"title" validate:"required"`
    Content string    `json:"content" gql:"content" grpc:"content"`
    Created time.Time `json:"created" gql:"created" grpc:"created"`
}

// Automatic endpoints:
// REST:      POST /notes, GET /notes, GET /notes/:id
// GraphQL:   query { notes { id title } }, mutation { createNote(...) }
// gRPC:      service NoteService { rpc Create(Note) returns (Note) }
// WebSocket: subscribe to notes { ... } (real-time updates)
```

**Benefits:**
- Frontend usa GraphQL para queries eficientes
- Backend microservices usan gRPC para performance
- Legacy clients siguen usando REST
- Todo desde una única definición

### 2. **Real-time by Default** ⚡

**Subscriptions automáticas:**
```go
// GraphQL subscription (auto-generated)
subscription {
  noteCreated {
    id
    title
  }
}

// WebSocket endpoint (auto-generated)
ws://api.example.com/notes/subscribe?filter={"where":{"userId":"123"}}
```

**Triggers:**
- `after_save` → broadcast to subscribers
- `after_delete` → notify disconnections
- Filter-aware (solo envía a clientes relevantes)

### 3. **AI-Native APIs** 🤖

**Vector Search Built-in:**
```go
type Document struct {
    westack.Model
    Content   string    `json:"content"`
    Embedding []float32 `json:"embedding" vector:"1536"` // Auto-indexed
}

// Automatic endpoint:
// POST /documents/search-semantic
// Body: { "query": "find similar documents", "k": 10 }
```

**Streaming Responses (LLM outputs):**
```go
model.BindStreamingOperation(AIModel, func(req Request) <-chan string {
    // Return channel for streaming LLM tokens
})

// Automatic SSE endpoint:
// GET /ai/generate?prompt=... (returns Server-Sent Events)
```

### 4. **Observability Native** 📊

**OpenTelemetry Integration:**
- Automatic traces para cada request (REST/GraphQL/gRPC)
- Custom spans en hooks y business logic
- Metrics: latency, error rate, throughput por modelo
- Structured logging con correlation IDs

**Ejemplo:**
```go
// Automatic tracing en hooks
func (n *Note) BeforeSave(ctx *westack.EventContext) error {
    // ctx includes trace span automatically
    span := ctx.Span()
    span.AddEvent("validating note")
    // ...
}
```

### 5. **Multi-Datasource Support** 💾

**Plugin Architecture:**
```go
// Config (declarativo)
datasources:
  primary:
    type: postgres
    connection: postgresql://...
  vector:
    type: weaviate
    connection: http://weaviate:8080
  cache:
    type: redis
    connection: redis://...

// Model usa datasource apropiado
type Note struct {
    westack.Model `datasource:"primary"`
    // ...
}

type Embedding struct {
    westack.Model `datasource:"vector"`
    // ...
}
```

**Supported (v3.0):**
- Postgres (via pgx)
- MongoDB (mantener v2)
- SQLite (edge-ready)
- Vector DBs: Weaviate, Pinecone, Qdrant
- Redis (caching/rate limiting)

### 6. **Event-Driven First** 📮

**Event Bus Opcional:**
```go
// Emit events automáticamente en hooks
func (n *Note) AfterSave(ctx *westack.EventContext) error {
    ctx.Emit("note.created", n) // Publish to event bus
    return nil
}

// Consume events en otro servicio
app.Subscribe("note.created", func(event westack.Event) {
    // Handle asynchronously
})

// Backends soportados: NATS, Kafka, Redis Streams, In-Memory
```

**Benefits:**
- Desacopla servicios
- CQRS support (command/query separation)
- Auditabilidad (event sourcing lite)

### 7. **Type-Safe Ecosystem** 🔒

**Client Generation:**
```bash
# Generate TypeScript client
westack-go generate client --lang=typescript --output=./frontend/src/api

# Generate Go client (for microservices)
westack-go generate client --lang=go --output=./client

# Generate Python client
westack-go generate client --lang=python --output=./sdk
```

**Benefits:**
- Frontend/backend share types (no drift)
- Autocomplete en IDEs
- Compile-time errors, no runtime surprises

---

## Migration Path: v2 → v3

### Phase 1: Compatibility Layer (v3.0)
**Timeline:** 3-6 meses

- ✅ v2 models siguen funcionando (JSON → Go auto-migration)
- ⚠️ Deprecation warnings para JSON models
- ✅ Coexistencia v2/v3 en mismo proyecto
- 🆕 New projects start with v3 pure-Go models

**Example:**
```go
// v2 style (deprecated pero funciona)
app.LoadModelsFromJSON("./models/*.json")

// v3 style (recommended)
app.RegisterModels(&Note{}, &User{})
```

### Phase 2: Feature Parity (v3.1)
**Timeline:** 6-12 meses

- ✅ All v2 features available en v3
- 🆕 GraphQL support (opt-in)
- 🆕 Real-time subscriptions (opt-in)
- 🆕 OpenTelemetry (opt-in)

### Phase 3: v2 EOL (v3.2+)
**Timeline:** 12+ meses

- ⚠️ JSON models removed
- ✅ Migration tool: `westack-go migrate v2-to-v3`
- ✅ Full v3 feature set

### Breaking Changes Summary

**Removed:**
- ❌ JSON model definitions (use Go structs)
- ❌ Fiber dependency (use stdlib net/http + chi/gorilla mux)
- ❌ Some legacy hooks (replaced with middleware)

**Changed:**
- 🔄 Hook signatures (add context.Context first param)
- 🔄 Model registration (explicit, not auto-discovery)
- 🔄 RBAC policies (extended for GraphQL/gRPC)

**Added:**
- ✅ Protocol adapters (GraphQL, gRPC, WebSocket, SSE)
- ✅ Event bus support
- ✅ Multi-datasource
- ✅ Observability
- ✅ AI patterns

---

## Competitive Positioning

| Feature | westack-go v3 | Hasura | Supabase | Ent | GORM |
|---------|---------------|--------|----------|-----|------|
| **Multi-protocol** | ✅ REST+GraphQL+gRPC | ❌ Solo GraphQL | ❌ Solo REST | ❌ Solo Go API | ❌ Solo Go API |
| **Real-time** | ✅ Built-in | ✅ Subscriptions | ✅ Real-time | ❌ Manual | ❌ Manual |
| **Type-safe** | ✅ Go native | ⚠️ Codegen | ⚠️ TypeScript | ✅ Go native | ⚠️ Reflection |
| **RBAC** | ✅ Casbin enterprise | ⚠️ Básico | ⚠️ Básico | ❌ Manual | ❌ Manual |
| **Self-hosted** | ✅ | ✅ | ⚠️ Complejo | ✅ | ✅ |
| **AI-ready** | ✅ Vector search | ❌ | ❌ | ❌ | ❌ |
| **Event-driven** | ✅ Optional | ❌ | ❌ | ❌ | ❌ |
| **Observability** | ✅ OpenTelemetry | ⚠️ Básico | ⚠️ Básico | ❌ | ❌ |

**Unique Selling Points:**
1. **Solo framework Go con multi-protocol from single source**
2. **Enterprise RBAC out-of-the-box** (no otros frameworks lo tienen)
3. **AI-native patterns** (vector search, streaming)
4. **Self-hosted + Cloud-ready** (no vendor lock-in)

---

## Implementation Priorities

### v3.0 MVP (Core)
**Timeline:** 3-6 meses

1. ✅ Pure Go models (eliminate JSON)
2. ✅ Multi-datasource (Postgres + MongoDB + SQLite)
3. ✅ GraphQL adapter (basic CRUD)
4. ✅ OpenTelemetry integration
5. ✅ v2 compatibility layer
6. ✅ Migration tool

### v3.1 (Real-time)
**Timeline:** 6-9 meses

1. ✅ WebSocket subscriptions
2. ✅ SSE streaming
3. ✅ Event bus (NATS/Redis)
4. ✅ GraphQL subscriptions

### v3.2 (AI & Advanced)
**Timeline:** 9-12 meses

1. ✅ Vector datasource plugins
2. ✅ Streaming responses (LLM)
3. ✅ gRPC adapter
4. ✅ Client generation (TS/Go/Python)

### v3.3+ (Enterprise)
**Timeline:** 12+ meses

1. ✅ Service mesh integration (Istio/Linkerd)
2. ✅ Circuit breakers
3. ✅ Distributed rate limiting
4. ✅ Multi-tenant patterns
5. ✅ GraphQL federation

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Scope creep** | 🔴 High | 🔴 High | Phased releases, MVP first |
| **Breaking v2 users** | 🟡 Medium | 🔴 High | Compatibility layer, migration tool |
| **Performance regression** | 🟡 Medium | 🟡 Medium | Benchmarks, profiling |
| **Community adoption** | 🟡 Medium | 🟡 Medium | Early beta, feedback loop |

### Market Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Hasura dominance** | 🟡 Medium | 🟡 Medium | Differentiate: multi-protocol + RBAC + self-hosted |
| **Ent/GORM entrenchment** | 🟡 Medium | 🔴 High | Focus on full-stack DX, not just ORM |
| **Framework fatigue** | 🟡 Medium | 🟡 Medium | Incremental migration, not revolution |

---

## Success Metrics

### Technical KPIs (v3.0)
- [ ] <50ms p99 latency (REST/GraphQL/gRPC)
- [ ] >90% test coverage
- [ ] Zero critical security issues
- [ ] <10MB binary size (minimal)

### Adoption KPIs (12 months)
- [ ] 1000+ GitHub stars (v3 release)
- [ ] 100+ production deployments
- [ ] 10+ community contributions
- [ ] 5+ enterprise users

### Developer Experience KPIs
- [ ] <5min from `westack-go init` to running API
- [ ] <1hr from tutorial to production deployment
- [ ] 95%+ positive feedback (surveys)

---

## Conclusion

**westack-go v3 debe ser:**
1. **Multi-protocol** - REST + GraphQL + gRPC + Real-time desde un único modelo
2. **Event-driven optional** - No forzado, pero first-class cuando se necesita
3. **AI-ready** - Vector search y streaming built-in para era LLM
4. **Observable** - OpenTelemetry nativo para debugging moderno
5. **Type-safe** - Pure Go, sin JSON indirection
6. **Backwards-compatible** - Migration path claro desde v2

**Philosophy:**  
*"Progressive enhancement, not revolution. Mantener la simplicidad de v2, agregar capacidades de 2025."*

**Next Steps:**
1. Community feedback (GitHub Discussions)
2. Prototype GraphQL adapter (proof of concept)
3. Benchmark multi-datasource performance
4. Design migration tool (JSON → Go)
5. RFC process para breaking changes

---

**Document Version:** 1.0  
**Created:** 2025-12-13  
**Authors:** Cascade AI + westack-go team  
**Status:** DRAFT - Awaiting community feedback
