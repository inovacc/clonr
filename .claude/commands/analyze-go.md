---
description: Analyze Go codebase for idiomatic patterns and best practices
---

# Go Code Analysis

Perform a comprehensive analysis of the Go codebase to identify non-idiomatic patterns and violations of Go best practices.

**IMPORTANT: This codebase requires Go 1.25.5** (check `.go-version` file)

1. **Verify Go version compliance**: Check `.go-version` and all `go.mod` files require Go 1.25.5
2. Scan the entire codebase (apps/ and shared/ directories, or as specified)
3. Identify violations of Go idiomatic patterns
4. Provide specific file paths and line numbers for each issue
5. Categorize issues by severity (Critical, High, Medium, Low)
6. Suggest concrete fixes with before/after code examples using Go 1.25.5 features
7. Create a prioritized action plan for improvements

Focus on:

- Package and file naming conventions
- Interface design patterns
- Error handling approaches
- Constant and variable naming
- Struct design and configuration management
- Dependency injection patterns
- Concurrency and goroutine management
- Logging consistency
- Code documentation

Provide a detailed report with actionable recommendations and code examples.
