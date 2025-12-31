# Go Code Analyzer - Quick Reference Guide

A concise reference for common Go anti-patterns and their idiomatic solutions.

## Quick Pattern Lookup

### Package & File Naming

| ❌ Anti-Pattern        | ✅ Idiomatic                          |
|-----------------------|--------------------------------------|
| `Immediate-charge/`   | `immediatecharge/`                   |
| `UserService/`        | `userservice/`                       |
| `my_package/`         | `mypackage/` (single word preferred) |
| `Immediate_charge.go` | `immediate_charge.go`                |
| `UserService.go`      | `user_service.go`                    |
| `type.go`             | `models.go` or `user.go` (specific)  |

### Interface Naming

| ❌ Anti-Pattern                    | ✅ Idiomatic                     |
|-----------------------------------|---------------------------------|
| `type StorageInterface interface` | `type Storage interface`        |
| `type IUserRepository interface`  | `type UserRepository interface` |
| `type AbstractHandler interface`  | `type Handler interface`        |

**Exception**: Single-method interfaces often use `-er` suffix:

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

### Constants

| ❌ Anti-Pattern          | ✅ Idiomatic               |
|-------------------------|---------------------------|
| `const MAX_SIZE = 100`  | `const MaxSize = 100`     |
| `const USER_ACTIVE = 1` | `const UserActive = 1`    |
| Untyped magic numbers   | Typed constants with iota |

**Idiomatic Pattern:**

```go
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusCompleted
)
```

### Error Handling

| ❌ Anti-Pattern                      | ✅ Idiomatic                              |
|-------------------------------------|------------------------------------------|
| `panic("error")` in libraries       | `return fmt.Errorf("error")`             |
| `return err` (loses context)        | `return fmt.Errorf("context: %w", err)`  |
| Ignoring errors: `value, _ := fn()` | Handle or explicitly comment why ignored |
| Generic error messages              | Specific, actionable errors              |

**Error Wrapping Pattern:**

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("do something: %w", err)
}
```

### Struct Design

| ❌ Anti-Pattern                  | ✅ Idiomatic                          |
|---------------------------------|--------------------------------------|
| Anonymous nested structs        | Named types                          |
| Public fields for internal data | Unexported fields with getters       |
| Embedding for "inheritance"     | Composition with explicit delegation |

**Anonymous Struct Problem:**

```go
// ❌ Bad
type Config struct {
    Server struct {
        Host string
        Port int
    }
}

// ✅ Good
type ServerConfig struct {
    Host string
    Port int
}

type Config struct {
    Server ServerConfig
}
```

### Concurrency

| ❌ Anti-Pattern                      | ✅ Idiomatic                 |
|-------------------------------------|-----------------------------|
| Naked goroutines                    | Use errgroup or WaitGroup   |
| `os.Exit` in goroutines             | Return errors to main       |
| No goroutine cleanup                | Proper context cancellation |
| Accessing shared state without sync | Mutexes or channels         |

**Goroutine Pattern:**

```go
// ❌ Bad
func Start() {
    go func() {
        if err := server.Serve(); err != nil {
            log.Fatal(err) // or os.Exit(1)
        }
    }()
}

// ✅ Good
func Start(ctx context.Context) error {
    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        return server.Serve()
    })

    return g.Wait()
}
```

### Dependency Injection

| ❌ Anti-Pattern                   | ✅ Idiomatic                       |
|----------------------------------|-----------------------------------|
| Constructor creates dependencies | Accept dependencies as parameters |
| Global state                     | Explicit dependency passing       |
| Service locator pattern          | Constructor injection             |

**Pattern:**

```go
// ❌ Bad
func NewService(cfg *Config) *Service {
    db := NewDatabase(cfg.DBURL) // Creates own deps
    return &Service{db: db}
}

// ✅ Good
func NewService(db Database) *Service {
    return &Service{db: db}
}
```

## Common Anti-Patterns by Category

### 🔴 Critical (Fix Immediately)

#### 1. Panic in Library Code

```go
// ❌ CRITICAL
func GetConfig() *Config {
    if config == nil {
        panic("not initialized")
    }
    return config
}

// ✅ FIXED
func GetConfig() (*Config, error) {
    if config == nil {
        return nil, errors.New("config not initialized")
    }
    return config, nil
}
```

#### 2. Race Conditions

```go
// ❌ CRITICAL - Race condition
type Cache struct {
    data map[string]string
    mu   sync.Mutex
}

func (c *Cache) Get(key string) string {
    return c.data[key] // Reading without lock!
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
}

// ✅ FIXED
type Cache struct {
    data map[string]string
    mu   sync.RWMutex
}

func (c *Cache) Get(key string) string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.data[key]
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
}
```

#### 3. os.Exit in Goroutines

```go
// ❌ CRITICAL
go func() {
    if err := run(); err != nil {
        os.Exit(1) // Abrupt termination!
    }
}()

// ✅ FIXED
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error {
    return run()
})
if err := g.Wait(); err != nil {
    return err
}
```

### 🟠 High Priority

#### 1. Missing Error Context

```go
// ❌ BAD
func ProcessUser(id int) error {
    user, err := db.GetUser(id)
    if err != nil {
        return err // Where did this fail?
    }

    if err := validator.Validate(user); err != nil {
        return err // What validation failed?
    }

    return nil
}

// ✅ GOOD
func ProcessUser(id int) error {
    user, err := db.GetUser(id)
    if err != nil {
        return fmt.Errorf("get user %d: %w", id, err)
    }

    if err := validator.Validate(user); err != nil {
        return fmt.Errorf("validate user %d: %w", id, err)
    }

    return nil
}
```

#### 2. Context Not Passed

```go
// ❌ BAD
func FetchData() ([]byte, error) {
    resp, err := http.Get("https://api.example.com/data")
    // No cancellation, no timeout
}

// ✅ GOOD
func FetchData(ctx context.Context) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    // Respects context cancellation/timeout
}
```

### 🟡 Medium Priority

#### 1. Inconsistent Naming

```go
// ❌ INCONSISTENT
type userService struct {}     // lowercase
type ProductRepository struct {} // PascalCase
type orderDAO struct {}        // abbrev DAO
const MAX_RETRIES = 3          // SCREAMING_SNAKE
const minTimeout = 1           // camelCase

// ✅ CONSISTENT
type userService struct {}
type productRepository struct {}
type orderRepository struct {} // Spelled out
const MaxRetries = 3
const MinTimeout = 1
```

#### 2. Bare Returns in Long Functions

```go
// ❌ CONFUSING
func Calculate(a, b int) (result int, err error) {
    result = a + b
    if result < 0 {
        err = errors.New("negative")
        return // What is being returned?
    }
    result *= 2
    return // What values?
}

// ✅ CLEAR
func Calculate(a, b int) (int, error) {
    result := a + b
    if result < 0 {
        return 0, errors.New("negative")
    }
    result *= 2
    return result, nil
}
```

### 🟢 Low Priority (Style)

#### 1. Comment Quality

```go
// ❌ BAD COMMENTS
// This function gets user
func GetUser(id int) (*User, error) {} // Obvious

// Crea un usuario nuevo (Spanish)
func CreateUser(user *User) error {}

// TODO: fix this (vague)
func Process() error {}

// ✅ GOOD COMMENTS
// GetUser retrieves a user by ID from the database.
// Returns ErrNotFound if the user doesn't exist.
func GetUser(id int) (*User, error) {}

// CreateUser persists a new user to the database after validation.
func CreateUser(user *User) error {}

// Process handles background job processing.
// TODO(john): Add retry logic for failed jobs (issue #123)
func Process() error {}
```

## Quick Decision Tree

### "Should I use a pointer receiver?"

```
Is the struct large (>few words)?
├─ Yes → Use pointer receiver
└─ No
   └─ Does the method modify the receiver?
      ├─ Yes → Use pointer receiver
      └─ No → Value receiver is fine
```

### "Should I return an error?"

```
Can this function fail?
├─ Yes → Return error
└─ No
   └─ Is it a constructor/initialization?
      ├─ Yes → Consider returning error anyway
      └─ No → No error return needed
```

### "Should I use a goroutine?"

```
Is this I/O bound or long-running?
├─ Yes
│  └─ Do I need to handle errors?
│     ├─ Yes → Use errgroup
│     └─ No → Use WaitGroup
└─ No → Don't use goroutine (overhead not worth it)
```

## Common Code Smells

### 1. "Util" or "Helper" Packages

```
❌ pkg/util/
❌ pkg/helpers/
✅ pkg/stringutil/  (specific purpose)
✅ pkg/mathext/     (specific purpose)
```

### 2. God Objects

```go
// ❌ 1000+ line service with 50 methods
type UserService struct {
    // Handles auth, profile, settings, notifications, etc.
}

// ✅ Focused services
type UserAuthService struct {}
type UserProfileService struct {}
type UserNotificationService struct {}
```

### 3. Premature Abstraction

```go
// ❌ Interface for single implementation
type UserRepository interface {
    Get(id int) (*User, error)
}

type MySQLUserRepository struct {} // Only implementation

// ✅ Concrete type first, interface when needed
type UserRepository struct {
    db *sql.DB
}
// Add interface later if second implementation appears
```

## Testing Patterns

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -1, -2},
        {"mixed", -1, 2, 1},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("got %d, want %d", result, tt.expected)
            }
        })
    }
}
```

## Go 1.25.5 Specific Features

### Enhanced Range Over Func (Go 1.23+)

```go
// Iterate over custom types
func (s *Set) All() func(yield func(int) bool) {
    return func(yield func(int) bool) {
        for v := range s.items {
            if !yield(v) {
                return
            }
        }
    }
}

// Usage
for v := range mySet.All() {
    fmt.Println(v)
}
```

### Improved Type Inference (Go 1.25+)

```go
// Better generic type inference
func Map[T, U any](slice []T, fn func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}

// Can often omit type parameters
result := Map(numbers, strconv.Itoa) // Types inferred
```

## Tools Quick Reference

### Run Multiple Linters

```bash
golangci-lint run --enable-all --disable=... ./...
```

### Find Race Conditions

```bash
go test -race ./...
```

### Check for Vulnerabilities

```bash
govulncheck ./...
```

### Format and Organize Imports

```bash
gofmt -w .
goimports -w .
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Go Proverbs](https://go-proverbs.github.io/)
- [50 Shades of Go](http://devs.cloudimmunity.com/gotchas-and-common-mistakes-in-go-golang/)
