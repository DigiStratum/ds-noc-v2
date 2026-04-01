#!/bin/bash
# check-tool-versions.sh
# Verify local tool versions match .tool-versions
#
# Usage:
#   ./scripts/check-tool-versions.sh           # Check all tools
#   ./scripts/check-tool-versions.sh --strict  # Fail on minor/patch mismatch
#
# Exit codes:
#   0 - All versions match (or within tolerance in default mode)
#   1 - Major version mismatch (always fails)
#   2 - Minor/patch mismatch (only fails in strict mode)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TOOL_VERSIONS_FILE="$REPO_ROOT/.tool-versions"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

STRICT_MODE=false
ERRORS=0
WARNINGS=0

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --strict)
            STRICT_MODE=true
            ;;
        --help|-h)
            echo "Usage: $0 [--strict]"
            echo ""
            echo "Verify local tool versions match .tool-versions"
            echo ""
            echo "Options:"
            echo "  --strict   Fail on any version mismatch (default: warn on minor/patch)"
            exit 0
            ;;
    esac
done

if [[ ! -f "$TOOL_VERSIONS_FILE" ]]; then
    echo -e "${YELLOW}Warning: .tool-versions not found, skipping version check${NC}"
    exit 0
fi

# Extract major.minor.patch from version string
parse_version() {
    local version="$1"
    # Handle versions like "go1.24.2", "v20.18.3", "1.24.2"
    echo "$version" | sed -E 's/^(go|v)?([0-9]+\.[0-9]+(\.[0-9]+)?).*/\2/'
}

# Compare two versions: returns "match", "major", "minor", or "patch"
compare_versions() {
    local expected="$1"
    local actual="$2"
    
    local exp_major exp_minor exp_patch
    local act_major act_minor act_patch
    
    IFS='.' read -r exp_major exp_minor exp_patch <<< "$expected"
    IFS='.' read -r act_major act_minor act_patch <<< "$actual"
    
    # Default patch to 0 if not specified
    exp_patch="${exp_patch:-0}"
    act_patch="${act_patch:-0}"
    
    if [[ "$exp_major" != "$act_major" ]]; then
        echo "major"
    elif [[ "$exp_minor" != "$act_minor" ]]; then
        echo "minor"
    elif [[ "$exp_patch" != "$act_patch" ]]; then
        echo "patch"
    else
        echo "match"
    fi
}

# Get installed version for a tool
get_installed_version() {
    local tool="$1"
    
    case "$tool" in
        golang)
            if command -v go &>/dev/null; then
                go version | sed -E 's/go version go([0-9]+\.[0-9]+\.[0-9]+).*/\1/'
            else
                echo ""
            fi
            ;;
        nodejs)
            if command -v node &>/dev/null; then
                node --version | sed 's/^v//'
            else
                echo ""
            fi
            ;;
        *)
            echo ""
            ;;
    esac
}

echo "Checking tool versions against .tool-versions..."
echo ""

# Read .tool-versions and check each tool
while IFS=' ' read -r tool expected_version || [[ -n "$tool" ]]; do
    # Skip comments and empty lines
    [[ -z "$tool" || "$tool" =~ ^# ]] && continue
    
    installed_version=$(get_installed_version "$tool")
    
    if [[ -z "$installed_version" ]]; then
        echo -e "${RED}✗ $tool: not installed (expected $expected_version)${NC}"
        ((ERRORS++))
        continue
    fi
    
    # Parse versions for comparison
    expected_parsed=$(parse_version "$expected_version")
    installed_parsed=$(parse_version "$installed_version")
    
    diff_level=$(compare_versions "$expected_parsed" "$installed_parsed")
    
    case "$diff_level" in
        match)
            echo -e "${GREEN}✓ $tool: $installed_version (matches $expected_version)${NC}"
            ;;
        major)
            echo -e "${RED}✗ $tool: $installed_version (expected $expected_version) - MAJOR version mismatch!${NC}"
            echo -e "  ${RED}This may cause go.mod to auto-upgrade. Please install the correct version.${NC}"
            ((ERRORS++))
            ;;
        minor|patch)
            if $STRICT_MODE; then
                echo -e "${RED}✗ $tool: $installed_version (expected $expected_version)${NC}"
                ((ERRORS++))
            else
                echo -e "${YELLOW}⚠ $tool: $installed_version (expected $expected_version) - minor/patch difference${NC}"
                ((WARNINGS++))
            fi
            ;;
    esac
done < "$TOOL_VERSIONS_FILE"

echo ""

# Summary and exit
if [[ $ERRORS -gt 0 ]]; then
    echo -e "${RED}Version check failed with $ERRORS error(s)${NC}"
    echo ""
    echo "To fix, install the correct versions using mise or asdf:"
    echo "  mise install    # or: asdf install"
    echo ""
    echo "Or update .tool-versions if upgrading intentionally."
    exit 1
elif [[ $WARNINGS -gt 0 ]]; then
    echo -e "${YELLOW}Version check passed with $WARNINGS warning(s)${NC}"
    echo "Run with --strict to treat warnings as errors."
    exit 0
else
    echo -e "${GREEN}All tool versions match!${NC}"
    exit 0
fi
