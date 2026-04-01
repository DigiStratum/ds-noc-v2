#!/bin/bash
# generate-ecosystems-yaml.sh
# Generate initial ecosystems.yaml from existing app configuration
#
# Usage:
#   ./scripts/generate-ecosystems-yaml.sh <app-directory>
#   ./scripts/generate-ecosystems-yaml.sh <app-directory> --dry-run
#   ./scripts/generate-ecosystems-yaml.sh <app-directory> -o /path/to/output.yaml
#
# Examples:
#   ./scripts/generate-ecosystems-yaml.sh ~/repos/digistratum/DSKanban
#   ./scripts/generate-ecosystems-yaml.sh ~/repos/digistratum/DSAccount --dry-run

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Defaults
DRY_RUN=false
OUTPUT_FILE=""
VERBOSE=false

usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <app-directory>

Generate initial ecosystems.yaml from existing app configuration.

Arguments:
  app-directory    Path to the app repository root

Options:
  --dry-run, -n    Preview output without writing file
  -o, --output     Output file path (default: <app-directory>/ecosystems.yaml)
  -v, --verbose    Show detailed extraction info
  -h, --help       Show this help message

Examples:
  $(basename "$0") ~/repos/digistratum/DSKanban
  $(basename "$0") ~/repos/digistratum/DSAccount --dry-run
  $(basename "$0") . -o ecosystems.yaml
EOF
    exit 0
}

log_info() {
    echo -e "${GREEN}✓${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $*" >&2
}

log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "  → $*" >&2
    fi
}

# Capitalize first letter of a string (portable)
capitalize_first() {
    local str="$1"
    echo "$(echo "${str:0:1}" | tr '[:lower:]' '[:upper:]')${str:1}"
}

# Extract app name from various sources
extract_app_name() {
    local app_dir="$1"
    local app_name=""
    
    # Priority 1: CDK stack appName property
    local cdk_stack_file
    for pattern in "infra/lib/*-stack.ts" "infra/lib/*Stack.ts"; do
        cdk_stack_file=$(find "$app_dir" -path "$app_dir/$pattern" 2>/dev/null | head -1)
        if [[ -n "$cdk_stack_file" ]]; then
            # Extract appName value from patterns like "appName: 'dskanban'," (with optional whitespace/comma)
            # Use awk for cleaner extraction
            app_name=$(grep -E "^\s*appName:\s*['\"]" "$cdk_stack_file" 2>/dev/null | head -1 | awk -F"['\"]" '{print $2}' || true)
            if [[ -n "$app_name" ]]; then
                log_verbose "Found appName in CDK stack: $app_name"
                echo "$app_name"
                return 0
            fi
        fi
    done
    
    # Priority 2: Root package.json name field
    if [[ -f "$app_dir/package.json" ]]; then
        app_name=$(jq -r '.name // empty' "$app_dir/package.json" 2>/dev/null | sed 's/@[^/]*\///' || true)
        if [[ -n "$app_name" ]]; then
            log_verbose "Found name in package.json: $app_name"
            echo "$app_name"
            return 0
        fi
    fi
    
    # Priority 3: Directory name
    app_name=$(basename "$(realpath "$app_dir")" | tr '[:upper:]' '[:lower:]')
    log_verbose "Using directory name: $app_name"
    echo "$app_name"
}

# Extract display name
extract_display_name() {
    local app_dir="$1"
    local app_name="$2"
    local display_name=""
    
    # Priority 1: package.json description
    if [[ -f "$app_dir/package.json" ]]; then
        display_name=$(jq -r '.description // empty' "$app_dir/package.json" 2>/dev/null || true)
        if [[ -n "$display_name" ]]; then
            # Clean up description to be a short display name
            # Take first sentence or first 50 chars
            display_name=$(echo "$display_name" | sed 's/\. .*//' | cut -c1-50)
            log_verbose "Found description in package.json: $display_name"
            echo "$display_name"
            return 0
        fi
    fi
    
    # Priority 2: README.md first heading
    if [[ -f "$app_dir/README.md" ]]; then
        display_name=$(head -5 "$app_dir/README.md" | grep -E "^#\s+" | head -1 | sed 's/^#\s*//' || true)
        if [[ -n "$display_name" ]]; then
            log_verbose "Found heading in README: $display_name"
            echo "$display_name"
            return 0
        fi
    fi
    
    # Priority 3: Title-case the app name
    display_name=$(echo "$app_name" | sed 's/ds/DS/' | sed 's/lk/LK/' | sed 's/-/ /g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) tolower(substr($i,2))}1')
    log_verbose "Generated from app name: $display_name"
    echo "$display_name"
}

# Extract SSO app ID
extract_sso_app_id() {
    local app_dir="$1"
    local app_name="$2"
    local sso_app_id=""
    
    # Priority 1: Environment config
    for env_file in "$app_dir/.env" "$app_dir/.env.local" "$app_dir/backend/.env"; do
        if [[ -f "$env_file" ]]; then
            sso_app_id=$(grep -E "^(SSO_APP_ID|APP_ID)=" "$env_file" 2>/dev/null | head -1 | cut -d= -f2 | tr -d "'\"" || true)
            if [[ -n "$sso_app_id" ]]; then
                log_verbose "Found SSO app ID in $env_file: $sso_app_id"
                echo "$sso_app_id"
                return 0
            fi
        fi
    done
    
    # Priority 2: CDK stack (SSO config) - look for ssoAppId: 'value' pattern
    local cdk_stack_file
    for pattern in "infra/lib/*-stack.ts" "infra/lib/*Stack.ts"; do
        cdk_stack_file=$(find "$app_dir" -path "$app_dir/$pattern" 2>/dev/null | head -1)
        if [[ -n "$cdk_stack_file" ]]; then
            # Use awk to extract value between quotes
            sso_app_id=$(grep -E "ssoAppId:\s*['\"]" "$cdk_stack_file" 2>/dev/null | head -1 | awk -F"['\"]" '{print $2}' || true)
            if [[ -n "$sso_app_id" ]]; then
                log_verbose "Found SSO app ID in CDK stack: $sso_app_id"
                echo "$sso_app_id"
                return 0
            fi
        fi
    done
    
    # Priority 3: Default to app name
    log_verbose "Using app name as SSO app ID: $app_name"
    echo "$app_name"
}

# Detect current ecosystem from domain config
detect_ecosystem() {
    local app_dir="$1"
    local ecosystem="digistratum"  # Default
    
    # Check CDK stack for domain patterns
    local cdk_stack_file
    for pattern in "infra/lib/*-stack.ts" "infra/lib/*Stack.ts"; do
        cdk_stack_file=$(find "$app_dir" -path "$app_dir/$pattern" 2>/dev/null | head -1)
        if [[ -n "$cdk_stack_file" ]]; then
            # Look for domain patterns
            if grep -qE "leapkick\.com" "$cdk_stack_file" 2>/dev/null; then
                ecosystem="leapkick"
                log_verbose "Detected leapkick ecosystem from CDK stack"
            elif grep -qE "digistratum\.com" "$cdk_stack_file" 2>/dev/null; then
                ecosystem="digistratum"
                log_verbose "Detected digistratum ecosystem from CDK stack"
            fi
        fi
    done
    
    echo "$ecosystem"
}

# Generate ecosystems.yaml content
generate_yaml() {
    local app_name="$1"
    local display_name="$2"
    local sso_app_id="$3"
    local ecosystem="$4"
    local ecosystem_capitalized
    ecosystem_capitalized=$(capitalize_first "$ecosystem")
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    cat << EOF
# ecosystems.yaml - Multi-ecosystem deployment configuration
#
# Generated by: generate-ecosystems-yaml.sh
# Generated at: ${timestamp}
#
# This file declares which ecosystems your app participates in.
# When present, CDK creates:
#   - {app}-data-{env}-{ecosystem} stacks (one per ecosystem-env)
#   - {app}-app-{env} stack (single CF distribution for all ecosystems)
#
# Without ecosystems.yaml, the app runs in legacy single-ecosystem mode.
# See: docs/ECOSYSTEMS.md for schema documentation and examples.

version: 1

app:
  # App name - used in stack names, resource names, and domain prefixes
  name: ${app_name}

  # Human-readable display name (for UI/logging)
  displayName: "${display_name}"

ecosystems:
  # ${ecosystem_capitalized} ecosystem (migrated from existing app)
  - name: ${ecosystem}
    enabled: true
    # SSO app ID registered with account.${ecosystem}.com
    sso_app_id: ${sso_app_id}

  # To add LeapKick ecosystem, uncomment and configure:
  # - name: leapkick
  #   enabled: true
  #   sso_app_id: ${app_name}
EOF
}

# Parse arguments
APP_DIR=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            ;;
        *)
            if [[ -z "$APP_DIR" ]]; then
                APP_DIR="$1"
            else
                log_error "Unexpected argument: $1"
                usage
            fi
            shift
            ;;
    esac
done

# Validate arguments
if [[ -z "$APP_DIR" ]]; then
    log_error "App directory is required"
    echo "" >&2
    usage
fi

# Resolve and validate app directory
APP_DIR=$(realpath "$APP_DIR" 2>/dev/null || echo "$APP_DIR")
if [[ ! -d "$APP_DIR" ]]; then
    log_error "Directory not found: $APP_DIR"
    exit 1
fi

# Check for app indicators
if [[ ! -f "$APP_DIR/package.json" ]] && [[ ! -d "$APP_DIR/infra" ]]; then
    log_error "Not a valid app directory: missing package.json and infra/"
    exit 1
fi

# Set default output file
if [[ -z "$OUTPUT_FILE" ]]; then
    OUTPUT_FILE="$APP_DIR/ecosystems.yaml"
fi

# Extract configuration
log_info "Analyzing app: $APP_DIR"

APP_NAME=$(extract_app_name "$APP_DIR")
if [[ -z "$APP_NAME" ]]; then
    log_error "Could not determine app name"
    exit 1
fi
log_info "App name: $APP_NAME"

DISPLAY_NAME=$(extract_display_name "$APP_DIR" "$APP_NAME")
log_info "Display name: $DISPLAY_NAME"

SSO_APP_ID=$(extract_sso_app_id "$APP_DIR" "$APP_NAME")
log_info "SSO app ID: $SSO_APP_ID"

ECOSYSTEM=$(detect_ecosystem "$APP_DIR")
log_info "Ecosystem: $ECOSYSTEM"

# Generate YAML
YAML_CONTENT=$(generate_yaml "$APP_NAME" "$DISPLAY_NAME" "$SSO_APP_ID" "$ECOSYSTEM")

# Output
echo "" >&2
if [[ "$DRY_RUN" == "true" ]]; then
    echo "=== DRY RUN - Would write to: $OUTPUT_FILE ===" >&2
    echo "" >&2
    echo "$YAML_CONTENT"
    echo "" >&2
    echo "=== END DRY RUN ===" >&2
else
    echo "$YAML_CONTENT" > "$OUTPUT_FILE"
    log_info "Generated: $OUTPUT_FILE"
fi
