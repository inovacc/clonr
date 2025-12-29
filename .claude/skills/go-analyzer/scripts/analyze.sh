#!/bin/bash

# Go Code Analyzer - Automated Analysis Script
# Usage: ./analyze.sh [directory] [options]
#
# This script runs comprehensive Go code analysis using standard tooling
# and generates a report following go-analyzer skill guidelines.

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TARGET_DIR="${1:-.}"
OUTPUT_DIR="analysis_$(date +%Y%m%d_%H%M%S)"
REQUIRED_GO_VERSION="1.25.5"
VERBOSE=${VERBOSE:-false}

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

section_header() {
    echo ""
    echo "======================================"
    echo "$1"
    echo "======================================"
}

# Check if required tools are installed
check_tools() {
    section_header "Checking Required Tools"

    local missing_tools=()

    if ! command -v go &> /dev/null; then
        missing_tools+=("go")
    fi

    if ! command -v golangci-lint &> /dev/null; then
        log_warning "golangci-lint not found (optional but recommended)"
        log_info "Install: https://golangci-lint.run/usage/install/"
    fi

    if ! command -v staticcheck &> /dev/null; then
        log_warning "staticcheck not found (optional)"
        log_info "Install: go install honnef.co/go/tools/cmd/staticcheck@latest"
    fi

    if ! command -v govulncheck &> /dev/null; then
        log_warning "govulncheck not found (optional)"
        log_info "Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
    fi

    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi

    log_success "Go is installed: $(go version)"
}

# Check Go version compliance
check_go_version() {
    section_header "Go Version Compliance Check"

    # Check .go-version file
    if [ -f ".go-version" ]; then
        local file_version=$(cat .go-version | tr -d '[:space:]')
        log_info ".go-version file: $file_version"

        if [ "$file_version" != "$REQUIRED_GO_VERSION" ]; then
            log_warning ".go-version ($file_version) doesn't match required ($REQUIRED_GO_VERSION)"
        else
            log_success ".go-version matches required version"
        fi
    else
        log_warning ".go-version file not found"
    fi

    # Check all go.mod files
    log_info "Checking go.mod files..."
    local mod_files=$(find "$TARGET_DIR" -name "go.mod" -not -path "*/vendor/*")
    local version_issues=0

    for mod_file in $mod_files; do
        local mod_version=$(grep "^go " "$mod_file" | awk '{print $2}')
        if [[ "$mod_version" != "$REQUIRED_GO_VERSION"* ]]; then
            log_warning "$(dirname $mod_file): go.mod version $mod_version"
            ((version_issues++))
        else
            log_success "$(dirname $mod_file): version OK"
        fi
    done

    echo "$version_issues" > "$OUTPUT_DIR/version_issues.txt"
}

# Run go vet
run_go_vet() {
    section_header "Running go vet"

    log_info "Analyzing with go vet..."
    if go vet ./... 2>&1 | tee "$OUTPUT_DIR/go_vet.txt"; then
        log_success "go vet passed"
    else
        log_warning "go vet found issues (see go_vet.txt)"
    fi
}

# Run golangci-lint
run_golangci_lint() {
    if ! command -v golangci-lint &> /dev/null; then
        log_warning "Skipping golangci-lint (not installed)"
        return
    fi

    section_header "Running golangci-lint"

    log_info "Running comprehensive linting..."

    # Create golangci-lint config if it doesn't exist
    if [ ! -f ".golangci.yml" ]; then
        log_info "Creating default .golangci.yml config..."
        cat > .golangci.yml << 'EOF'
run:
  timeout: 5m
  go: "1.25"

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unparam
    - unconvert
    - gocritic
    - revive

linters-settings:
  errcheck:
    check-blank: true
  gocritic:
    enabled-tags:
      - diagnostic
      - style
      - performance
  revive:
    rules:
      - name: exported
        arguments:
          - "checkPrivateReceivers"
          - "sayRepetitiveInsteadOfStutters"

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
EOF
    fi

    if golangci-lint run --out-format colored-line-number ./... 2>&1 | tee "$OUTPUT_DIR/golangci_lint.txt"; then
        log_success "golangci-lint passed"
    else
        log_warning "golangci-lint found issues (see golangci_lint.txt)"
    fi
}

# Run staticcheck
run_staticcheck() {
    if ! command -v staticcheck &> /dev/null; then
        log_warning "Skipping staticcheck (not installed)"
        return
    fi

    section_header "Running staticcheck"

    log_info "Analyzing with staticcheck..."
    if staticcheck ./... 2>&1 | tee "$OUTPUT_DIR/staticcheck.txt"; then
        log_success "staticcheck passed"
    else
        log_warning "staticcheck found issues (see staticcheck.txt)"
    fi
}

# Run govulncheck
run_govulncheck() {
    if ! command -v govulncheck &> /dev/null; then
        log_warning "Skipping govulncheck (not installed)"
        return
    fi

    section_header "Running govulncheck"

    log_info "Checking for known vulnerabilities..."
    if govulncheck ./... 2>&1 | tee "$OUTPUT_DIR/govulncheck.txt"; then
        log_success "No vulnerabilities found"
    else
        log_error "Vulnerabilities detected! (see govulncheck.txt)"
    fi
}

# Run tests with race detector
run_race_detector() {
    section_header "Running Race Detector"

    log_info "Running tests with -race flag..."
    if go test -race -short ./... 2>&1 | tee "$OUTPUT_DIR/race_detector.txt"; then
        log_success "No race conditions detected"
    else
        log_error "Race conditions found! (see race_detector.txt)"
    fi
}

# Analyze package naming
analyze_package_naming() {
    section_header "Analyzing Package Naming"

    log_info "Checking for non-idiomatic package names..."

    local issues_found=0

    # Find directories with Go files (packages)
    while IFS= read -r pkg_dir; do
        local pkg_name=$(basename "$pkg_dir")

        # Check for kebab-case
        if [[ "$pkg_name" =~ - ]]; then
            echo "❌ Kebab-case: $pkg_dir" | tee -a "$OUTPUT_DIR/naming_issues.txt"
            ((issues_found++))
        fi

        # Check for uppercase
        if [[ "$pkg_name" =~ [A-Z] ]]; then
            echo "❌ Contains uppercase: $pkg_dir" | tee -a "$OUTPUT_DIR/naming_issues.txt"
            ((issues_found++))
        fi

    done < <(find "$TARGET_DIR" -type f -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -exec dirname {} \; | sort -u)

    if [ $issues_found -eq 0 ]; then
        log_success "All package names follow Go conventions"
    else
        log_warning "Found $issues_found package naming issues (see naming_issues.txt)"
    fi
}

# Search for common anti-patterns
find_antipatterns() {
    section_header "Searching for Anti-Patterns"

    log_info "Looking for common Go anti-patterns..."

    # Panic in non-main packages
    log_info "Checking for panic() in library code..."
    if grep -r "panic(" "$TARGET_DIR" --include="*.go" --exclude-dir=vendor --exclude-dir=.git \
        | grep -v "_test.go" | grep -v "cmd/\|main.go" > "$OUTPUT_DIR/panic_usage.txt" 2>/dev/null; then
        log_warning "Found panic() in library code (see panic_usage.txt)"
    else
        log_success "No problematic panic() usage"
    fi

    # Interface suffix
    log_info "Checking for 'Interface' suffix..."
    if grep -r "type.*Interface interface" "$TARGET_DIR" --include="*.go" --exclude-dir=vendor \
        > "$OUTPUT_DIR/interface_suffix.txt" 2>/dev/null; then
        log_warning "Found interfaces with 'Interface' suffix (see interface_suffix.txt)"
    else
        log_success "No 'Interface' suffix found"
    fi

    # SCREAMING_SNAKE_CASE constants
    log_info "Checking for SCREAMING_SNAKE_CASE constants..."
    if grep -r "const.*[A-Z_]*[A-Z]_[A-Z]" "$TARGET_DIR" --include="*.go" --exclude-dir=vendor \
        > "$OUTPUT_DIR/snake_case_constants.txt" 2>/dev/null; then
        log_warning "Found SCREAMING_SNAKE_CASE constants (see snake_case_constants.txt)"
    else
        log_success "No SCREAMING_SNAKE_CASE constants found"
    fi

    # os.Exit in goroutines
    log_info "Checking for os.Exit in goroutines..."
    if grep -B 3 "os.Exit" "$TARGET_DIR" --include="*.go" --exclude-dir=vendor -r \
        | grep -B 3 "go func" > "$OUTPUT_DIR/os_exit_in_goroutine.txt" 2>/dev/null; then
        log_error "Found os.Exit in goroutines! (see os_exit_in_goroutine.txt)"
    else
        log_success "No os.Exit in goroutines"
    fi
}

# Generate summary report
generate_report() {
    section_header "Generating Summary Report"

    local report_file="$OUTPUT_DIR/REPORT.md"

    cat > "$report_file" << EOF
# Go Code Analysis Report

**Generated**: $(date)
**Target**: $TARGET_DIR
**Go Version Required**: $REQUIRED_GO_VERSION

## 🔧 Tool Execution Summary

EOF

    # Add tool results
    for tool_output in "$OUTPUT_DIR"/*.txt; do
        if [ -f "$tool_output" ]; then
            local tool_name=$(basename "$tool_output" .txt)
            local line_count=$(wc -l < "$tool_output")

            if [ $line_count -eq 0 ]; then
                echo "- ✅ **$tool_name**: No issues" >> "$report_file"
            else
                echo "- ⚠️  **$tool_name**: $line_count items found" >> "$report_file"
            fi
        fi
    done

    cat >> "$report_file" << EOF

## 📁 Detailed Results

All detailed results are available in the output directory:
\`$OUTPUT_DIR\`

### Files Generated:
EOF

    ls -1 "$OUTPUT_DIR" | grep -v "REPORT.md" | while read -r file; do
        echo "- \`$file\`" >> "$report_file"
    done

    cat >> "$report_file" << EOF

## 🎯 Next Steps

1. Review files marked with ⚠️  or ✗
2. Prioritize fixes based on severity (vulnerabilities > race conditions > style)
3. Use the go-analyzer skill for detailed guidance on fixing anti-patterns
4. Run this script again after fixes to verify improvements

## 📚 References

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
EOF

    log_success "Report generated: $report_file"

    # Display summary
    echo ""
    cat "$report_file"
}

# Main execution
main() {
    section_header "Go Code Analyzer"
    echo "Target: $TARGET_DIR"
    echo "Output: $OUTPUT_DIR"
    echo ""

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Change to target directory
    cd "$TARGET_DIR"

    # Run all checks
    check_tools
    check_go_version
    run_go_vet
    run_golangci_lint
    run_staticcheck
    run_govulncheck
    run_race_detector
    analyze_package_naming
    find_antipatterns
    generate_report

    section_header "Analysis Complete!"
    echo "Results saved to: $OUTPUT_DIR"
    echo ""
    echo "To view the full report:"
    echo "  cat $OUTPUT_DIR/REPORT.md"
    echo ""
}

# Run main function
main "$@"
