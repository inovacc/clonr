---
name: go-expert
description: Expert guidance for Go development, architecture, patterns, and best practices. Use when implementing Go features, designing systems, or seeking idiomatic Go solutions.
project_types: [go]
activation: [manual, auto]
priority: medium
auto_triggers: [implement, create, design, build, develop, architecture, pattern, refactor]
---

# Go Expert

**IMPORTANT: Always respond in English, regardless of the language used in the question.**

Comprehensive expertise in Go programming, covering modern idioms, architectural patterns, performance optimization, and production-ready implementations using Go 1.25+.

## When to Use This Skill

This skill should be used when:

- Implementing new Go features or services
- Designing Go application architecture
- Seeking idiomatic Go solutions to problems
- Optimizing Go code for performance
- Making technology choices for Go projects
- Writing production-ready Go code
- Troubleshooting complex Go issues
- Implementing concurrency patterns
- Working with Go generics, interfaces, and type systems

## Core Capabilities

### 1. Modern Go Development (Go 1.25+)

Expert guidance on Go 1.25.5 features and modern Go idioms.

**Key Areas:**

- **Generics**: Type parameters, constraints, and type inference
- **Enhanced standard library**: `log/slog`, improved `testing`, `slices`, `maps` packages
- **Range over functions**: Iterator patterns with `range` keyword
- **Improved error handling**: Error wrapping, `errors.Join()`, multi-error patterns
- **Context patterns**: Proper context propagation and cancellation

**Example - Range over functions (Go 1.23+):**

```go
// Iterator function that yields values
func Fibonacci(max int) func(func(int) bool) {
    return func(yield func(int) bool) {
        a, b := 0, 1
        for a < max {
            if !yield(a) {
                return
            }
            a, b = b, a+b
        }
    }
}

// Usage with range
for n := range Fibonacci(100) {
    fmt.Println(n)
}
```

### 2. Application Architecture

Designing scalable, maintainable Go applications.

**Architectural Patterns:**

- **Clean Architecture**: Domain-driven design with Go
- **Hexagonal Architecture**: Ports and adapters pattern
- **CQRS**: Command Query Responsibility Segregation
- **Event-Driven**: Message brokers, pub/sub patterns
- **Microservices**: Service communication, API design

**Example - Clean Architecture structure:**

```
project/
├── cmd/                    # Application entrypoints
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/            # Business logic (entities, use cases)
│   │   ├── user.go
│   │   └── user_service.go
│   ├── adapters/          # External integrations
│   │   ├── http/          # HTTP handlers
│   │   ├── postgres/      # Database implementations
│   │   └── redis/         # Cache implementations
│   └── ports/             # Interfaces
│       ├── repositories.go
│       └── services.go
├── pkg/                   # Public libraries
└── go.mod
```

### 3. Concurrency Patterns

Expert implementation of Go's concurrency primitives.

**Patterns:**

- **Worker Pools**: Bounded concurrency with channels
- **Pipeline**: Multi-stage concurrent processing
- **Fan-out/Fan-in**: Parallel processing with result aggregation
- **Context Cancellation**: Graceful shutdown and timeout handling
- **Errgroup**: Coordinated goroutine error handling

**Example - Worker pool with errgroup:**

```go
import "golang.org/x/sync/errgroup"

func ProcessItems(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10) // Max 10 concurrent workers

    for _, item := range items {
        item := item // Go 1.22+ doesn't need this, but shown for clarity
        g.Go(func() error {
            return processItem(ctx, item)
        })
    }

    return g.Wait() // Returns first error or nil
}
```

### 4. Performance Optimization

Identifying and resolving performance bottlenecks.

**Techniques:**

- **Profiling**: CPU, memory, goroutine, and mutex profiling
- **Benchmarking**: Writing effective benchmarks
- **Memory optimization**: Reducing allocations, pooling
- **Algorithm selection**: Choosing efficient data structures
- **Caching strategies**: In-memory, distributed caching

**Example - Memory pooling with sync.Pool:**

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func ProcessData(data []byte) ([]byte, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer for processing
    buf.Write(data)
    return buf.Bytes(), nil
}
```

### 5. Testing Strategies

Writing comprehensive, maintainable tests.

**Testing Types:**

- **Unit tests**: Table-driven tests, mocking, test fixtures
- **Integration tests**: Database, HTTP, external service testing
- **Benchmark tests**: Performance regression detection
- **Fuzz tests**: Random input testing (Go 1.18+)
- **E2E tests**: Full application flow testing

**Example - Table-driven test:**

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        want    bool
        wantErr bool
    }{
        {"valid email", "user@example.com", true, false},
        {"missing @", "userexample.com", false, true},
        {"empty", "", false, true},
        {"multiple @", "user@@example.com", false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("wantErr %v, got %v", tt.wantErr, err)
            }
            if got != tt.want {
                t.Errorf("want %v, got %v", tt.want, got)
            }
        })
    }
}
```

## Configuration Templates

Production-ready configuration files are available in the **go-analyzer** skill:

### `.editorconfig` and `.golangci.yml`

Located at `skills/go-analyzer/`

These comprehensive configuration files provide:

- **`.editorconfig`**: Consistent formatting (tabs for Go, proper line endings, 120-char limit)
- **`.golangci.yml`**: 40+ linters with detailed documentation and rationale

**Usage:**

```bash
# Copy configuration files from go-analyzer
cp skills/go-analyzer/.editorconfig /path/to/your/project/
cp skills/go-analyzer/.golangci.yml /path/to/your/project/

# Run linter
golangci-lint run ./...
```

**Why consolidated in go-analyzer?**
These configs are primarily for code analysis and quality checking, so they're maintained in the go-analyzer skill to avoid duplication. Both skills reference the same standards.

## Best Practices

### 1. Error Handling

- Always handle errors explicitly
- Use `%w` for error wrapping
- Return errors, don't panic in libraries
- Use `errors.Is()` and `errors.As()` for error checking

```go
// ✅ Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ❌ Bad
if err := doSomething(); err != nil {
    panic(err) // Don't panic in library code
}
```

### 2. Interface Design

- Keep interfaces small (1-3 methods)
- Accept interfaces, return structs
- Define interfaces where they're used, not where they're implemented
- Use `-er` suffix for single-method interfaces

```go
// ✅ Good - defined in consumer package
type UserGetter interface {
    GetUser(id string) (*User, error)
}

// ❌ Bad - overly large interface
type UserService interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
    UpdateUser(user *User) error
    DeleteUser(id string) error
    ListUsers(filter Filter) ([]*User, error)
    // ... 10 more methods
}
```

### 3. Dependency Injection

- Use constructor functions
- Inject dependencies explicitly
- Avoid global state
- Use interfaces for testing

```go
// ✅ Good - explicit dependencies
type Service struct {
    db     Database
    cache  Cache
    logger *slog.Logger
}

func NewService(db Database, cache Cache, logger *slog.Logger) *Service {
    return &Service{db: db, cache: cache, logger: logger}
}
```

### 4. Context Usage

- Pass context as first parameter
- Don't store context in structs
- Use context for cancellation, deadlines, and request-scoped values
- Always check `ctx.Err()` in long-running operations

```go
// ✅ Good
func (s *Service) FetchData(ctx context.Context, id string) (*Data, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        return s.db.Query(ctx, id)
    }
}
```

### 5. Structured Logging

- Use `log/slog` (Go 1.21+) for structured logging
- Include context (request ID, user ID, etc.)
- Use appropriate log levels
- Don't log sensitive data

```go
// ✅ Good - structured logging with slog
logger.InfoContext(ctx, "user created",
    slog.String("user_id", user.ID),
    slog.String("email", user.Email),
    slog.Duration("elapsed", time.Since(start)),
)
```

## Anti-Patterns to Avoid

### ❌ Goroutine Leaks

```go
// Bad - goroutine never stops
func StartWorker() {
    go func() {
        for {
            work := <-workChan
            process(work)
        }
    }()
}

// ✅ Good - proper shutdown
func StartWorker(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case work := <-workChan:
                process(work)
            }
        }
    }()
}
```

### ❌ Premature Optimization

```go
// Bad - complex optimization before measuring
func (s *Service) Get(id string) *User {
    // 100 lines of caching, pooling, etc.
}

// ✅ Good - start simple, optimize when needed
func (s *Service) Get(ctx context.Context, id string) (*User, error) {
    return s.db.GetUser(ctx, id)
}
```

### ❌ Naked Returns

```go
// Bad - unclear what's being returned
func calculate(a, b int) (result int, err error) {
    result = a + b
    return // What's being returned?
}

// ✅ Good - explicit returns
func calculate(a, b int) (int, error) {
    result := a + b
    return result, nil
}
```

## Tools & Resources

### Recommended Versions (December 2025)

**Core Tools:**

- **Go**: 1.25.5 (current stable)
- **golangci-lint**: v1.62+
- **staticcheck**: v0.6+
- **govulncheck**: latest

**Development:**

- **gopls**: Language server (latest)
- **delve**: Debugger (latest)
- **air**: Live reload (v1.52+)

**Installation:**

```bash
# Install Go 1.25.5
go install golang.org/dl/go1.25.5@latest
go1.25.5 download

# Install tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Useful Packages

**HTTP/Web:**

- `net/http` - Standard library HTTP server
- `chi` - Lightweight router
- `fiber` - High-performance web framework

**Database:**

- `database/sql` - Standard database interface
- `pgx` - PostgreSQL driver and toolkit
- `sqlc` - Generate type-safe code from SQL

**Configuration:**

- `viper` - Configuration management
- `envconfig` - Environment variable parsing

**Observability:**

- `opentelemetry-go` - Distributed tracing
- `prometheus/client_golang` - Metrics

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Go Blog](https://go.dev/blog/)
- [Go Wiki](https://github.com/golang/go/wiki)
