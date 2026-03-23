# Go Coding Standards & Best Practices

A veteran Go architect's guide to building well-architected, performant, and maintainable solutions. Rooted in SOLID, DRY, and CLEAN principles applied idiomatically to Go.

## 1. Architecture Layers

```
Entry Point (CLI/HTTP) → Service → Repository → Database
```

**Each layer has exactly one job:**

| Layer | Does | Does NOT |
|-------|------|----------|
| **Entry point** | Parse input, call service, format output | Contain logic, touch DB, validate business rules |
| **Service** | Orchestrate, validate, enforce rules, own transactions | Format output, parse CLI args, know about HTTP |
| **Repository** | CRUD, queries, prepared statements | Calculate, derive, filter in Go, enforce business rules |
| **Model** | Carry data, structural validation | Import services, know about persistence |

**Violation smell:** If you're importing `repository` from a CLI command, or importing `fmt` for user output in a service, you've crossed a boundary.

---

## 2. Dependency Injection

**Go doesn't need a DI framework.** Constructor injection is explicit, compile-safe, and readable.

```go
// Constructor declares what it needs. No magic.
func NewOrderService(
    repo OrderRepository,    // interface, not *SqlOrderRepository
    payment PaymentGateway,  // interface
    notify NotificationSender, // interface
) *OrderService {
    return &OrderService{repo: repo, payment: payment, notify: notify}
}
```

**Rules:**
- Accept interfaces, return structs
- Define interfaces at the **consumer**, not the implementor
- Keep interfaces small (1-3 methods). Split `ReadWriter` into `Reader` + `Writer`
- No `init()` side effects, no package-level mutable state, no service locators
- Wire everything in `main()` or a dedicated `wire.go` — nowhere else

**Anti-pattern:** Global singletons, `init()` that registers services, `GetInstance()` methods.

---

## 3. Thin Controllers (CLI Commands / HTTP Handlers)

A controller is **three lines of logic:**

```go
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var req CreateOrderRequest                          // 1. Parse
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, err)
        return
    }

    order, err := h.orderSvc.Create(r.Context(), req.ToInput()) // 2. Delegate
    if err != nil {
        respondError(w, mapError(err))
        return
    }

    respondJSON(w, http.StatusCreated, order)           // 3. Respond
}
```

**If your handler has a `for` loop, an `if` that checks business state, or imports a repository — it's too fat.**

Same for CLI:
```go
func runCreateOrder(cmd *cobra.Command, args []string) error {
    input := parseFlags(cmd)                    // 1. Parse
    order, err := svc.Create(cmd.Context(), input) // 2. Delegate
    if err != nil { return err }
    formatOutput(order)                         // 3. Format
    return nil
}
```

---

## 4. DTOs and Data Flow

```
Request DTO → Service Input → Domain Model → Repository → Domain Model → Response DTO
```

| Type | Purpose | Lives in |
|------|---------|----------|
| **Request DTO** | Parse + structural validation of external input | handler/command package |
| **Service Input** | Business-level params, decoupled from transport | service package |
| **Domain Model** | Core entity, business invariants | models package |
| **Response DTO** | Shape output, hide internals (no DB IDs) | handler/command package |

**Rules:**
- Domain models never have `json:"..."` tags driven by API needs (use a response DTO)
- Service inputs use business names (`OrderKey`, not `id int64`)
- DTOs are **dumb structs** — no methods beyond `Validate()` or `ToInput()`
- Never pass a Cobra `*cobra.Command` or `http.Request` into a service

---

## 5. Error Handling

```go
// Wrap with context at every layer boundary
return fmt.Errorf("creating order for customer %s: %w", custID, err)
```

**Rules:**
- Always wrap. Never `return err` naked across a boundary
- Use sentinel errors for expected conditions: `var ErrNotFound = errors.New("not found")`
- Use typed errors when callers need to branch: `type ValidationError struct { Field, Msg string }`
- Check with `errors.Is()` / `errors.As()` — never string-match
- Never ignore errors with `_`. If truly ignorable, comment why
- Panics are bugs. Recover only at the top-level entry point

**Exit code mapping belongs in the entry point, not the service:**
```go
// handler or CLI — translates domain errors to transport codes
var notFound *NotFoundError
if errors.As(err, &notFound) {
    return http.StatusNotFound  // or os.Exit(1)
}
```

---

## 6. Context

```go
// First param. Always. No exceptions.
func (s *OrderService) Cancel(ctx context.Context, key string) error
```

- Never store `context.Context` in a struct
- Use for cancellation, deadlines, and request-scoped values only
- Pass through to every DB call, HTTP call, and downstream service
- Don't use context for dependency injection (that's what constructors are for)

---

## 7. Repository Pattern

```go
// Interface defined by the consumer (service package)
type OrderRepository interface {
    Create(ctx context.Context, order *Order) error
    GetByKey(ctx context.Context, key string) (*Order, error)
    UpdateStatus(ctx context.Context, id int64, status Status) error
}
```

**Rules:**
- Repositories do **one table** (or one aggregate root)
- No joins that span aggregates — coordinate in the service
- Accept `*sql.Tx` when participating in a service-owned transaction
- Return domain models, not `sql.Row`
- Use parameterized queries. Always. No string concatenation
- Repository methods are **leaf operations** — they don't call other repositories

---

## 8. Transaction Ownership

**Services own transactions. Repositories participate.**

```go
func (s *TransferService) Execute(ctx context.Context, from, to string, amount int) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return fmt.Errorf("begin tx: %w", err) }
    defer tx.Rollback()

    if err := s.accountRepo.DebitTx(ctx, tx, from, amount); err != nil { return err }
    if err := s.accountRepo.CreditTx(ctx, tx, to, amount); err != nil { return err }
    if err := s.ledgerRepo.RecordTx(ctx, tx, from, to, amount); err != nil { return err }

    return tx.Commit()
}
```

A repository never calls `BeginTx`. It receives `*sql.Tx` and operates within it.

---

## 9. Testing Strategy

| Layer | Test with | Speed |
|-------|-----------|-------|
| Repository | Real DB, cleanup before each test | Slow (integration) |
| Service | Mocked interfaces | Fast (unit) |
| Handler/CLI | Mocked services | Fast (unit) |
| Model | Nothing (pure logic) | Instant |

**Mock pattern — function fields, no framework:**
```go
type MockOrderRepo struct {
    GetByKeyFunc func(ctx context.Context, key string) (*Order, error)
}
func (m *MockOrderRepo) GetByKey(ctx context.Context, key string) (*Order, error) {
    return m.GetByKeyFunc(ctx, key)
}
```

**Rules:**
- Table-driven tests for anything with >2 cases
- Test behavior, not implementation
- Every error path gets a test
- Repository tests clean up their own data — never rely on test ordering
- No `time.Sleep` in tests. Use channels, contexts, or polling

---

## 10. Package Design

```
internal/
  models/       # Domain types. Zero dependencies on other internal packages
  repository/   # Data access. Depends on: models
  services/     # Business logic. Depends on: models, repository interfaces
  cli/          # CLI entry. Depends on: services (via interfaces)
  api/          # HTTP entry. Depends on: services (via interfaces)
```

**Rules:**
- No circular imports. Ever. If you need one, your boundary is wrong
- `internal/` for everything that isn't a public API
- One package = one concept. `utils` is a code smell — distribute helpers to where they're used
- Package names are lowercase, single-word, noun-based: `order`, `payment`, `auth`
- Don't stutter: `order.Order` is fine; `order.OrderService` is fine; `order.OrderOrderer` is not

---

## 11. SOLID in Go

| Principle | Go Interpretation |
|-----------|-------------------|
| **S** — Single Responsibility | One struct, one reason to change. Service doesn't format output. Repo doesn't validate |
| **O** — Open/Closed | Extend via interfaces and composition, not by modifying existing code |
| **L** — Liskov Substitution | Any implementation of an interface must be substitutable without surprises |
| **I** — Interface Segregation | Small interfaces. `io.Reader` has one method. Yours should too, when possible |
| **D** — Dependency Inversion | Services depend on interfaces (abstractions), not concrete repos |

---

## 12. Performance & Concurrency

- **Prepared statements** for repeated queries
- **Connection pooling** — configure `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`
- **WAL mode** for SQLite (reads don't block writes)
- **`sync.Once`** for expensive one-time initialization
- **No premature goroutines.** Sequential code is easier to reason about. Only parallelize when you've measured a bottleneck
- **Always select on `ctx.Done()`** in long-running goroutines
- **Prefer channels for communication, mutexes for state.** If you're using both, simplify

---

## 13. Code Hygiene

- `gofmt` is non-negotiable. Run it on save
- `golangci-lint` catches what `go vet` misses
- Named return values only when they clarify (e.g., `(n int, err error)` in `io.Reader`), never just for naked returns
- Exported names get doc comments. Period
- Constants over magic numbers: `const maxRetries = 3`, not `for i := 0; i < 3; i++`
- Early returns over deep nesting:
  ```go
  // Good
  if err != nil { return err }
  // proceed...

  // Bad
  if err == nil {
      // 40 lines of indented code
  }
  ```

---

## 14. Configuration

- Read config at startup, pass values down via constructors
- Never read env vars deep in business logic
- Use a config struct, not scattered `os.Getenv()` calls
- Validate config at startup — fail fast, not at 3am when a handler first reads a missing var

---

## 15. What "Done" Looks Like

Before any code is considered complete:

```bash
make fmt && make lint && make test
```

All three pass. No warnings suppressed. No tests skipped. No `//nolint` without a comment explaining why.

---

**The throughline:** Every decision optimizes for the next developer reading this code in 6 months. That developer might be you. Be kind to them.
