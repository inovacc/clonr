---
name: go-analyzer
description: Analyzes Go codebase for idiomatic patterns and best practices. Use when analyzing Go code quality, identifying non-idiomatic patterns, or checking against Go conventions.
project_types: [go]
activation: [manual, auto]
priority: high
auto_triggers: [analyze, review, check, audit, quality, lint, validate]
---

# Go Code Analyzer

**IMPORTANT: Always respond in English, regardless of the language used in the question.**

A comprehensive skill for analyzing Go codebases to identify non-idiomatic patterns and violations of best practices.

## When to Use This Skill

This skill should be used when:

- Analyzing Go code for quality issues
- Reviewing for idiomatic Go patterns
- Checking code against Go conventions
- Identifying technical debt in Go projects
- Preparing code for production review

## Go Version Requirements

**Required Go Version: 1.25.5** (as specified in `.go-version`)

Before analyzing, verify:

1. Check that `go.mod` specifies `go 1.25.5` or compatible
2. Ensure code uses Go 1.25+ features appropriately
3. Flag any deprecated patterns from older Go versions
4. Verify no usage of pre-1.25 workarounds when native features exist

## Integration with Go Tooling

This skill provides comprehensive code analysis that complements automated tools. The analysis can be enhanced by using:

### Recommended Tools (December 2025)

**Static Analysis:**
- **golangci-lint** (v1.62+): Meta-linter aggregating multiple linters
  ```bash
  # Install
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

  # Run
  golangci-lint run ./...
  ```

- **staticcheck** (v0.6+): Advanced static analysis tool
  ```bash
  # Install
  go install honnef.co/go/tools/cmd/staticcheck@latest

  # Run
  staticcheck ./...
  ```

**Security & Vulnerabilities:**
- **govulncheck** (latest): Check for known vulnerabilities
  ```bash
  # Install
  go install golang.org/x/vuln/cmd/govulncheck@latest

  # Run
  govulncheck ./...
  ```

**Built-in Tools:**
- **go vet**: Official Go static analyzer
- **go test -race**: Race condition detector
- **gofmt/goimports**: Code formatting

### Configuration Templates

This skill includes production-ready configuration files that you can copy to your Go projects:

#### `.editorconfig`
Ensures consistent code formatting across editors and IDEs. Located at `skills/go-analyzer/.editorconfig`.

**Key settings:**
- **Go files**: Uses tabs (gofmt standard)
- **YAML/JSON**: 2-space indentation
- **Shell scripts**: 2-space indentation (Google style)
- **Line length**: 120 characters (configurable)
- **Line endings**: LF (Unix-style)

**Usage:**
```bash
# Copy to your project root
cp skills/go-analyzer/.editorconfig /path/to/your/project/

# Supported by VS Code, IntelliJ, Sublime, Vim, and more
# No additional setup needed - editors detect it automatically
```

#### `.golangci.yml`
Comprehensive golangci-lint configuration with 40+ linters enabled. Located at `skills/go-analyzer/.golangci.yml`.

**Features:**
- **Organized by category**: Style, Security, Performance, Testing, etc.
- **Detailed comments**: Every linter explained with rationale
- **Balanced approach**: Strict on bugs/security, flexible on style
- **Production-ready**: Based on industry best practices

**Enabled linters include:**
- `govet`, `errcheck`, `staticcheck` - Core static analysis
- `bodyclose` - Ensures HTTP response bodies are closed
- `copyloopvar` - Detects loop variable capture (Go 1.22+)
- `gocritic` - Meta-linter with 100+ checks
- `ineffassign` - Finds ineffectual assignments
- `unused` - Detects unused code

**Disabled (with explanations):**
- `gosec` - Run separately for security audits (many false positives)
- `lll` - Line length handled by EditorConfig
- `dupl` - Code duplication isn't always bad
- `godox` - TODOs are useful during development

**Usage:**
```bash
# Copy to your project root
cp skills/go-analyzer/.golangci.yml /path/to/your/project/

# Run golangci-lint with the configuration
golangci-lint run ./...

# Or customize for your project's needs
# Edit .golangci.yml to enable/disable specific linters
```

**Integration with this skill:**
1. Copy configuration files to your project
2. Run `golangci-lint run ./...` for automated checks
3. Use this skill for in-depth architectural analysis
4. Combine results for comprehensive code quality

### Automated Analysis Script

Use the provided automation script for comprehensive analysis:

```bash
# Run full analysis
./scripts/analyze.sh

# Analyze specific directory
./scripts/analyze.sh ./apps/my-service

# View results
cat analysis_*/REPORT.md
```

### How This Skill Adds Value

While automated tools catch syntax and pattern issues, this skill provides:

1. **Architectural Analysis**: Evaluates design patterns and project structure
2. **Contextual Recommendations**: Suggests fixes based on Go 1.25.5 features
3. **Prioritized Action Plans**: Identifies high-impact improvements
4. **Best Practice Guidance**: Explains WHY patterns are non-idiomatic
5. **Custom Pattern Detection**: Identifies project-specific anti-patterns

**Workflow:**
1. Run automated tools (golangci-lint, staticcheck) for quick wins
2. Use this skill for in-depth architectural and pattern analysis
3. Combine results for comprehensive code quality improvements

## Analysis Areas

### 1. Package Structure & Naming

- Check for kebab-case or PascalCase in package names (should be lowercase)
- Verify directory names match Go conventions
- Check for inconsistent naming patterns

**Example Issue:**

```
❌ apps/cob/internal/Immediate-charge/
✅ apps/cob/internal/immediatecharge/
```

### 2. Interface Naming

- Identify interfaces with redundant 'Interface' suffix
- Check for overly generic interface names
- Verify interface naming follows Go conventions (-er suffix when appropriate)

**Example Issue:**

```go
❌ type MessageBrokerInterface interface { Publish(...) }
✅ type MessageBroker interface { Publish(...) }
```

### 3. Error Handling

- Look for panic() usage in library code (should return errors)
- Check for proper error wrapping with %w
- Verify context.Context is used for cancellation

**Example Issue:**

```go
❌ func Get() *Config {
    if cfg == nil { panic("not loaded") }
    return cfg
}

✅ func Get() (*Config, error) {
    if cfg == nil { return nil, fmt.Errorf("not loaded") }
    return cfg, nil
}
```

### 4. Constants

- Check for SCREAMING_SNAKE_CASE (non-idiomatic)
- Look for untyped constants that should use iota
- Verify constant names are descriptive and properly scoped

**Example Issue:**

```go
❌ const (
    DRAFT = 0
    ACTIVE = 1
    BLOCKED = 2
)

✅ type AccountStatus int
const (
    AccountStatusDraft AccountStatus = iota
    AccountStatusActive
    AccountStatusBlocked
)
```

### 5. Struct Design

- Identify anonymous nested structs (should be named types)
- Check struct tag usage and consistency
- Verify exported vs unexported fields are appropriate

**Example Issue:**

```go
❌ type Config struct {
    Database struct {
        Host string
        Port int
    }
}

✅ type DatabaseConfig struct {
    Host string
    Port int
}
type Config struct {
    Database DatabaseConfig
}
```

### 6. Dependency Injection

- Look for constructors that create their own dependencies
- Verify proper dependency injection patterns
- Check for tight coupling issues

**Example Issue:**

```go
❌ func NewService(cfg *Config) (*Service, error) {
    client := NewClient(cfg.URL)  // Creates own deps
    return &Service{client: client}, nil
}

✅ func NewService(client Client) *Service {
    return &Service{client: client}
}
```

### 7. Concurrency

- Check for goroutines without proper error handling
- Verify use of sync primitives (WaitGroup, errgroup, etc.)
- Look for potential race conditions

**Example Issue:**

```go
❌ go func() {
    if err := server.Serve(); err != nil {
        os.Exit(1)  // os.Exit in goroutine
    }
}()

✅ g, ctx := errgroup.WithContext(ctx)
g.Go(func() error {
    return server.Serve()
})
if err := g.Wait(); err != nil {
    os.Exit(1)
}
```

### 8. File Naming

- Check for non-standard file names (should be lowercase with underscores)
- Identify overly generic names like 'type.go' or 'util.go'
- Verify test files use _test.go suffix

**Example Issue:**

```
❌ Immediate_charge.go (mixed case)
❌ type.go (too generic)
✅ immediate_charge.go
✅ interfaces.go or models.go
```

### 9. Logging

- Check for inconsistent logging approaches
- Verify structured logging is used
- Identify mixed logging libraries

**Example Issue:**

```go
❌ Mixed: logrus.Debug(...) and logger.Error(...)
✅ Consistent: slog.Info(...) throughout
```

### 10. Comments & Documentation

- Check for non-English comments
- Verify exported functions have proper godoc comments
- Look for outdated or misleading comments

**Example Issue:**

```go
❌ // ChargeDueInBatch define las operaciones...
✅ // ChargeDueInBatch defines operations for batch charge management.
```

## Analysis Process

When performing analysis:

1. **Verify Go version**: Check `.go-version` and `go.mod` for Go 1.25.5
2. **Scan directories**: Focus on apps/ and shared/ by default (or as specified)
3. **Identify violations**: Look for patterns listed above
4. **Categorize severity**:
  - 🔴 **Critical**: panic in library, race conditions, goroutine leaks, wrong Go version
  - 🟠 **High**: improper error handling, missing context, tight coupling
  - 🟡 **Medium**: naming conventions, interface design
  - 🟢 **Low**: documentation, file naming

5. **Provide locations**: Use `file_path:line_number` format
6. **Suggest fixes**: Show before/after code examples using Go 1.25.5 features
7. **Prioritize**: Quick wins vs long-term refactoring

## Output Format

Structure your analysis as:

```markdown
# Go Code Analysis Report

## 🔧 Go Version Check
**Required**: Go 1.25.5 (from `.go-version`)
**go.mod**: [version found]
**Status**: ✅ Compatible / ❌ Needs Update

## 📈 Executive Summary
Total Issues: 47
├─ 🔴 Critical: 3
├─ 🟠 High: 12
├─ 🟡 Medium: 24
└─ 🟢 Low: 8

## 🔍 Detailed Findings

### 🔴 Critical Issues

#### 1. Panic in Library Code
**Location**: `shared/config/config.go:23`
**Severity**: Critical

❌ **Problem**:
[code showing issue]

✅ **Solution**:
[code showing fix]

**Impact**: High - Can crash application unexpectedly
**Effort**: Low - Simple refactor

[Continue for each issue...]

## ✅ Positive Patterns

Things already following Go best practices:
- ✅ Context propagation throughout
- ✅ Circuit breaker implementation
- ✅ Error wrapping with %w

## 📋 Prioritized Action Plan

### Quick Wins (Low effort, High impact)
1. Rename interfaces (remove Interface suffix)
2. Fix constant naming
3. Translate comments to English

### Critical Fixes (Must do)
1. Replace panic with error returns
2. Implement errgroup for goroutines
3. Fix dependency injection in constructors

### Long-term Refactoring
1. Standardize on slog for logging
2. Extract anonymous structs in config
3. Consolidate naming conventions
```

## Best Practices for This Skill

- **Verify Go version first**: Always check `.go-version` and enforce Go 1.25.5
- **Be thorough but practical**: Focus on issues that genuinely impact code quality
- **Provide context**: Explain WHY something is non-idiomatic
- **Show examples**: Always include before/after code using Go 1.25.5 features
- **Consider exceptions**: Note when non-idiomatic code might be justified
- **Balance priorities**: Distinguish quick wins from major refactoring
- **Be specific**: Always include file paths and line numbers
- **Stay current**: Reference Go 1.25.5 features when relevant (slog, enhanced generics, ranges over func)

## References

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Go Proverbs](https://go-proverbs.github.io/)
