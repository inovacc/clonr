#!/bin/bash
# Project Type Detection Script
# Detects the type of project in the current directory
# Usage: ./detect-project.sh [directory]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Directory to analyze (default: current directory)
TARGET_DIR="${1:-.}"

# Change to target directory
cd "$TARGET_DIR" 2>/dev/null || {
    echo -e "${RED}Error: Directory '$TARGET_DIR' not found${NC}"
    exit 1
}

echo -e "${BLUE}🔍 Detecting project type in: $(pwd)${NC}"
echo ""

# Track detected types (a project can be multiple types)
DETECTED_TYPES=()

# ============================================
# Go Project Detection
# ============================================
detect_go() {
    local go_score=0
    local go_indicators=()

    # Strong indicators
    if [[ -f "go.mod" ]]; then
        go_score=$((go_score + 10))
        go_indicators+=("✓ go.mod found")
    fi

    if [[ -f ".go-version" ]]; then
        go_score=$((go_score + 5))
        go_indicators+=("✓ .go-version found")
    fi

    # Count .go files
    go_files=$(find . -maxdepth 3 -name "*.go" 2>/dev/null | wc -l)
    if [[ $go_files -gt 0 ]]; then
        go_score=$((go_score + go_files))
        go_indicators+=("✓ $go_files Go source files")
    fi

    # Check for go.sum
    if [[ -f "go.sum" ]]; then
        go_score=$((go_score + 3))
        go_indicators+=("✓ go.sum found")
    fi

    # Check for common Go directories
    if [[ -d "cmd" ]] || [[ -d "internal" ]] || [[ -d "pkg" ]]; then
        go_score=$((go_score + 5))
        go_indicators+=("✓ Go project structure (cmd/internal/pkg)")
    fi

    if [[ $go_score -ge 10 ]]; then
        DETECTED_TYPES+=("go")
        echo -e "${GREEN}✓ Go Project Detected${NC} (confidence: $go_score)"
        printf '  %s\n' "${go_indicators[@]}"
        echo ""

        # Extract Go version if available
        if [[ -f "go.mod" ]]; then
            go_version=$(grep -m 1 "^go " go.mod | awk '{print $2}')
            echo -e "  ${BLUE}Go Version:${NC} $go_version"
            echo ""
        fi

        return 0
    fi

    return 1
}

# ============================================
# Android Project Detection
# ============================================
detect_android() {
    local android_score=0
    local android_indicators=()

    # Strong indicators
    if [[ -f "build.gradle" ]] || [[ -f "build.gradle.kts" ]]; then
        android_score=$((android_score + 8))
        android_indicators+=("✓ Gradle build file found")
    fi

    if [[ -f "settings.gradle" ]] || [[ -f "settings.gradle.kts" ]]; then
        android_score=$((android_score + 5))
        android_indicators+=("✓ Gradle settings found")
    fi

    # Check for AndroidManifest.xml
    manifest_count=$(find . -name "AndroidManifest.xml" 2>/dev/null | wc -l)
    if [[ $manifest_count -gt 0 ]]; then
        android_score=$((android_score + 10))
        android_indicators+=("✓ $manifest_count AndroidManifest.xml found")
    fi

    # Check for Android-specific directories
    if [[ -d "app/src/main" ]]; then
        android_score=$((android_score + 7))
        android_indicators+=("✓ Android app structure")
    fi

    # Check for Kotlin files (common in modern Android)
    kt_files=$(find . -maxdepth 4 -name "*.kt" 2>/dev/null | wc -l)
    if [[ $kt_files -gt 0 ]]; then
        android_score=$((android_score + 3))
        android_indicators+=("✓ $kt_files Kotlin files")
    fi

    # Check for gradle wrapper
    if [[ -f "gradlew" ]]; then
        android_score=$((android_score + 3))
        android_indicators+=("✓ Gradle wrapper present")
    fi

    # Check for res directory
    if [[ -d "app/src/main/res" ]] || [[ -d "src/main/res" ]]; then
        android_score=$((android_score + 5))
        android_indicators+=("✓ Android resources directory")
    fi

    if [[ $android_score -ge 15 ]]; then
        DETECTED_TYPES+=("android")
        echo -e "${GREEN}✓ Android Project Detected${NC} (confidence: $android_score)"
        printf '  %s\n' "${android_indicators[@]}"
        echo ""

        # Try to detect Android SDK version
        if [[ -f "app/build.gradle" ]]; then
            target_sdk=$(grep "targetSdk" app/build.gradle 2>/dev/null | head -1 | grep -o '[0-9]\+' | head -1)
            min_sdk=$(grep "minSdk" app/build.gradle 2>/dev/null | head -1 | grep -o '[0-9]\+' | head -1)
            if [[ -n "$target_sdk" ]]; then
                echo -e "  ${BLUE}Target SDK:${NC} $target_sdk"
            fi
            if [[ -n "$min_sdk" ]]; then
                echo -e "  ${BLUE}Min SDK:${NC} $min_sdk"
            fi
            echo ""
        fi

        return 0
    fi

    return 1
}

# ============================================
# Python Project Detection
# ============================================
detect_python() {
    local python_score=0
    local python_indicators=()

    # Strong indicators
    if [[ -f "pyproject.toml" ]]; then
        python_score=$((python_score + 10))
        python_indicators+=("✓ pyproject.toml found")
    fi

    if [[ -f "requirements.txt" ]]; then
        python_score=$((python_score + 7))
        python_indicators+=("✓ requirements.txt found")
    fi

    if [[ -f "setup.py" ]]; then
        python_score=$((python_score + 7))
        python_indicators+=("✓ setup.py found")
    fi

    if [[ -f "Pipfile" ]]; then
        python_score=$((python_score + 7))
        python_indicators+=("✓ Pipfile found")
    fi

    # Count .py files
    py_files=$(find . -maxdepth 3 -name "*.py" 2>/dev/null | wc -l)
    if [[ $py_files -gt 0 ]]; then
        python_score=$((python_score + py_files))
        python_indicators+=("✓ $py_files Python files")
    fi

    # Check for common Python directories
    if [[ -d "venv" ]] || [[ -d ".venv" ]] || [[ -d "env" ]]; then
        python_score=$((python_score + 3))
        python_indicators+=("✓ Virtual environment directory")
    fi

    if [[ -f ".python-version" ]]; then
        python_score=$((python_score + 3))
        python_indicators+=("✓ .python-version found")
    fi

    if [[ $python_score -ge 10 ]]; then
        DETECTED_TYPES+=("python")
        echo -e "${GREEN}✓ Python Project Detected${NC} (confidence: $python_score)"
        printf '  %s\n' "${python_indicators[@]}"
        echo ""

        # Try to detect Python version
        if [[ -f ".python-version" ]]; then
            py_version=$(cat .python-version)
            echo -e "  ${BLUE}Python Version:${NC} $py_version"
            echo ""
        fi

        return 0
    fi

    return 1
}

# ============================================
# Run Detection
# ============================================

detect_go
detect_android
detect_python

# ============================================
# Summary
# ============================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [[ ${#DETECTED_TYPES[@]} -eq 0 ]]; then
    echo -e "${YELLOW}⚠ No recognized project type detected${NC}"
    echo ""
    echo "This directory does not appear to contain a Go, Android, or Python project."
    echo "Supported project indicators:"
    echo "  • Go: go.mod, .go-version, *.go files"
    echo "  • Android: build.gradle, AndroidManifest.xml, app/ structure"
    echo "  • Python: pyproject.toml, requirements.txt, setup.py, *.py files"
    echo ""
    exit 1
fi

echo -e "${GREEN}✓ Project Type(s):${NC} ${DETECTED_TYPES[*]}"
echo ""

# Recommend skills based on detected types
echo -e "${BLUE}📚 Recommended Claude Code Skills:${NC}"
echo ""

for project_type in "${DETECTED_TYPES[@]}"; do
    case $project_type in
        go)
            echo -e "  ${GREEN}Go Project:${NC}"
            echo "    • go-analyzer   - Code quality analysis, best practices review"
            echo "    • go-expert     - Development guidance, architecture, implementation"
            ;;
        android)
            echo -e "  ${GREEN}Android Project:${NC}"
            echo "    • android-senior-dev - Feature development, architecture, Compose"
            echo "    • android-forensic   - APK analysis, debugging, security audit"
            ;;
        python)
            echo -e "  ${GREEN}Python Project:${NC}"
            echo "    • python-expert - Development guidance, best practices (⚠ In Development)"
            ;;
    esac
done

echo ""
echo -e "${BLUE}💡 Tip:${NC} Skills will auto-activate based on your questions!"
echo ""

exit 0
