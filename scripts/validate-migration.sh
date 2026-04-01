#!/bin/bash
# validate-migration.sh
# Compare old vs new stack outputs and validate migration correctness
#
# Usage:
#   ./scripts/validate-migration.sh <app-name> <environment> [options]
#   ./scripts/validate-migration.sh <app-name> <environment> --ecosystem <ecosystem>
#   ./scripts/validate-migration.sh <app-name> <environment> --sample-size 20
#
# Examples:
#   ./scripts/validate-migration.sh dskanban prod
#   ./scripts/validate-migration.sh myapp dev --ecosystem digistratum
#   ./scripts/validate-migration.sh myapp prod --verbose --sample-size 50
#
# Prerequisites:
#   - AWS CLI configured with appropriate permissions
#   - Both old and new stacks deployed
#   - dig command available for DNS validation

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Defaults
VERBOSE=false
SAMPLE_SIZE=10
ECOSYSTEM=""
SKIP_DNS=false
SKIP_DATA=false

# AWS region (default to current)
AWS_REGION="${AWS_REGION:-$(aws configure get region 2>/dev/null || echo 'us-east-1')}"

# Validation results
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <app-name> <environment>

Compare old vs new stack outputs and validate migration correctness.

Arguments:
  app-name         Application name (e.g., 'dskanban', 'myapp')
  environment      Environment name: 'dev' or 'prod'

Options:
  --ecosystem, -e <name>  Ecosystem to validate (default: auto-detect from ecosystems.yaml)
  --sample-size, -s <n>   Number of records to spot-check (default: 10)
  --skip-dns              Skip DNS resolution checks
  --skip-data             Skip DynamoDB data validation
  -v, --verbose           Show detailed operation info
  -h, --help              Show this help message

Validation Checks:
  1. CloudFormation Outputs  - Compare key outputs between old/new stacks
  2. DNS Resolution          - Verify domains resolve to correct CloudFront
  3. DynamoDB Record Counts  - Compare source vs target table counts
  4. Data Integrity          - Spot-check sample records match

Stack Naming Conventions:
  Old (legacy):     {app}-{env}                    e.g., myapp-prod
  New (app):        {app}-app-{env}                e.g., myapp-app-prod
  New (data):       {app}-data-{env}-{ecosystem}   e.g., myapp-data-prod-digistratum

Examples:
  $(basename "$0") dskanban prod                      # Full validation
  $(basename "$0") myapp dev --verbose                # With detailed output
  $(basename "$0") myapp prod --skip-data             # Skip data checks (faster)
  $(basename "$0") myapp prod -e digistratum -s 50    # Specific ecosystem, 50 samples

Exit Codes:
  0   All checks passed
  1   One or more checks failed
  2   Usage/argument error
EOF
    exit 0
}

log_info() {
    echo -e "${GREEN}✓${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $*" >&2
    ((WARNINGS++)) || true
}

log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

log_step() {
    echo -e "${BLUE}→${NC} $*" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "  ${BLUE}·${NC} $*" >&2
    fi
}

log_header() {
    echo -e "\n${BOLD}$*${NC}" >&2
    echo -e "${BOLD}$(printf '=%.0s' $(seq 1 ${#1}))${NC}" >&2
}

pass_check() {
    local name="$1"
    ((TOTAL_CHECKS++)) || true
    ((PASSED_CHECKS++)) || true
    log_info "PASS: $name"
}

fail_check() {
    local name="$1"
    local reason="${2:-}"
    ((TOTAL_CHECKS++)) || true
    ((FAILED_CHECKS++)) || true
    if [[ -n "$reason" ]]; then
        log_error "FAIL: $name - $reason"
    else
        log_error "FAIL: $name"
    fi
}

warn_check() {
    local name="$1"
    local reason="${2:-}"
    ((TOTAL_CHECKS++)) || true
    ((PASSED_CHECKS++)) || true
    if [[ -n "$reason" ]]; then
        log_warn "WARN: $name - $reason"
    else
        log_warn "WARN: $name"
    fi
}

# Parse arguments
parse_args() {
    local positional=()
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -e|--ecosystem)
                ECOSYSTEM="$2"
                shift 2
                ;;
            -s|--sample-size)
                SAMPLE_SIZE="$2"
                shift 2
                ;;
            --skip-dns)
                SKIP_DNS=true
                shift
                ;;
            --skip-data)
                SKIP_DATA=true
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                echo "Use --help for usage information" >&2
                exit 2
                ;;
            *)
                positional+=("$1")
                shift
                ;;
        esac
    done
    
    if [[ ${#positional[@]} -lt 2 ]]; then
        log_error "Missing required arguments: app-name and environment"
        echo "Use --help for usage information" >&2
        exit 2
    fi
    
    APP_NAME="${positional[0]}"
    ENVIRONMENT="${positional[1]}"
    
    if [[ ! "$ENVIRONMENT" =~ ^(dev|prod)$ ]]; then
        log_error "Environment must be 'dev' or 'prod', got: $ENVIRONMENT"
        exit 2
    fi
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."
    
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLI not found. Please install and configure it."
        exit 2
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq not found. Please install it (brew install jq)."
        exit 2
    fi
    
    if [[ "$SKIP_DNS" == "false" ]] && ! command -v dig &> /dev/null; then
        log_warn "dig not found. DNS checks will be skipped."
        SKIP_DNS=true
    fi
    
    # Verify AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS credentials not configured or invalid."
        exit 2
    fi
    
    log_verbose "Prerequisites OK"
}

# Get stack outputs as JSON
get_stack_outputs() {
    local stack_name="$1"
    
    aws cloudformation describe-stacks \
        --stack-name "$stack_name" \
        --query "Stacks[0].Outputs" \
        --output json 2>/dev/null || echo "[]"
}

# Check if stack exists
stack_exists() {
    local stack_name="$1"
    aws cloudformation describe-stacks \
        --stack-name "$stack_name" &> /dev/null
}

# Get output value by key
get_output_value() {
    local outputs="$1"
    local key="$2"
    echo "$outputs" | jq -r ".[] | select(.OutputKey == \"$key\") | .OutputValue // empty"
}

# Validate CloudFormation outputs
validate_cf_outputs() {
    log_header "CloudFormation Output Comparison"
    
    local old_stack="${APP_NAME}-${ENVIRONMENT}"
    local new_app_stack="${APP_NAME}-app-${ENVIRONMENT}"
    local new_data_stack="${APP_NAME}-data-${ENVIRONMENT}"
    
    if [[ -n "$ECOSYSTEM" ]]; then
        new_data_stack="${APP_NAME}-data-${ENVIRONMENT}-${ECOSYSTEM}"
    fi
    
    # Check old stack exists
    if ! stack_exists "$old_stack"; then
        warn_check "Old stack check" "Stack $old_stack not found (may already be cleaned up)"
        return
    fi
    
    # Check new stacks exist
    if ! stack_exists "$new_app_stack"; then
        fail_check "New app stack existence" "Stack $new_app_stack not found"
        return
    fi
    
    log_verbose "Old stack: $old_stack"
    log_verbose "New app stack: $new_app_stack"
    
    local old_outputs
    local new_outputs
    
    old_outputs=$(get_stack_outputs "$old_stack")
    new_outputs=$(get_stack_outputs "$new_app_stack")
    
    if [[ "$old_outputs" == "[]" ]]; then
        warn_check "Old stack outputs" "No outputs found for $old_stack"
        return
    fi
    
    if [[ "$new_outputs" == "[]" ]]; then
        fail_check "New stack outputs" "No outputs found for $new_app_stack"
        return
    fi
    
    # Compare key outputs
    local key_outputs=("ApiUrl" "CloudFrontDomain" "CloudFrontDistributionId" "FrontendBucket")
    
    for key in "${key_outputs[@]}"; do
        local old_val new_val
        old_val=$(get_output_value "$old_outputs" "$key")
        new_val=$(get_output_value "$new_outputs" "$key")
        
        if [[ -n "$old_val" && -n "$new_val" ]]; then
            log_verbose "$key: old=$old_val new=$new_val"
            pass_check "Output exists: $key"
        elif [[ -n "$old_val" && -z "$new_val" ]]; then
            fail_check "Output migration: $key" "Present in old stack but missing in new"
        elif [[ -z "$old_val" && -n "$new_val" ]]; then
            pass_check "New output: $key"
        fi
    done
    
    # Report output counts
    local old_count new_count
    old_count=$(echo "$old_outputs" | jq 'length')
    new_count=$(echo "$new_outputs" | jq 'length')
    
    log_verbose "Old stack outputs: $old_count, New stack outputs: $new_count"
    
    if [[ "$new_count" -ge "$old_count" ]]; then
        pass_check "Output count" "New stack has equal or more outputs ($new_count >= $old_count)"
    else
        warn_check "Output count" "New stack has fewer outputs ($new_count < $old_count)"
    fi
}

# Validate DNS resolution
validate_dns() {
    if [[ "$SKIP_DNS" == "true" ]]; then
        log_verbose "DNS validation skipped"
        return
    fi
    
    log_header "DNS Resolution Validation"
    
    # Try to get CloudFront domain from new stack
    local new_app_stack="${APP_NAME}-app-${ENVIRONMENT}"
    local new_outputs
    new_outputs=$(get_stack_outputs "$new_app_stack")
    
    local cf_domain
    cf_domain=$(get_output_value "$new_outputs" "CloudFrontDomain")
    
    if [[ -z "$cf_domain" ]]; then
        log_verbose "CloudFront domain not found in stack outputs"
        cf_domain=$(get_output_value "$new_outputs" "DistributionDomainName")
    fi
    
    if [[ -z "$cf_domain" ]]; then
        warn_check "CloudFront domain lookup" "Could not determine CloudFront domain from stack outputs"
        return
    fi
    
    log_verbose "Expected CloudFront domain: $cf_domain"
    
    # Build list of domains to check
    local domains=()
    local base_domain
    
    # Determine base domain from environment
    if [[ "$ENVIRONMENT" == "prod" ]]; then
        base_domain="digistratum.com"
        domains+=("${APP_NAME}.${base_domain}")
    else
        base_domain="dev.digistratum.com"
        domains+=("${APP_NAME}.${base_domain}")
    fi
    
    # Add ecosystem domain if specified
    if [[ -n "$ECOSYSTEM" && "$ECOSYSTEM" != "digistratum" ]]; then
        if [[ "$ENVIRONMENT" == "prod" ]]; then
            domains+=("${APP_NAME}.${ECOSYSTEM}.com")
        else
            domains+=("${APP_NAME}.dev.${ECOSYSTEM}.com")
        fi
    fi
    
    for domain in "${domains[@]}"; do
        log_verbose "Checking DNS for: $domain"
        
        local resolved
        resolved=$(dig +short "$domain" CNAME 2>/dev/null | head -1 | sed 's/\.$//')
        
        if [[ -z "$resolved" ]]; then
            # Try A record (might be aliased)
            resolved=$(dig +short "$domain" A 2>/dev/null | head -1)
            if [[ -n "$resolved" ]]; then
                # Check if it resolves to CloudFront IPs
                local cf_ips
                cf_ips=$(dig +short "$cf_domain" A 2>/dev/null | head -1)
                if [[ "$resolved" == "$cf_ips" ]]; then
                    pass_check "DNS resolution: $domain" "A record matches CloudFront"
                else
                    warn_check "DNS resolution: $domain" "A record exists but may not point to CloudFront"
                fi
            else
                fail_check "DNS resolution: $domain" "No CNAME or A record found"
            fi
        elif [[ "$resolved" == "$cf_domain" || "$resolved" == *".cloudfront.net" ]]; then
            pass_check "DNS resolution: $domain" "Points to CloudFront"
        else
            fail_check "DNS resolution: $domain" "Points to $resolved, expected CloudFront"
        fi
    done
}

# Get DynamoDB table record count
get_table_count() {
    local table_name="$1"
    
    aws dynamodb describe-table \
        --table-name "$table_name" \
        --query "Table.ItemCount" \
        --output text 2>/dev/null || echo "0"
}

# Sample records from DynamoDB table
sample_records() {
    local table_name="$1"
    local limit="$2"
    
    aws dynamodb scan \
        --table-name "$table_name" \
        --max-items "$limit" \
        --output json 2>/dev/null || echo '{"Items":[]}'
}

# Check if record exists in target table
record_exists_in_target() {
    local target_table="$1"
    local key_json="$2"
    
    aws dynamodb get-item \
        --table-name "$target_table" \
        --key "$key_json" \
        --output json 2>/dev/null | jq -e '.Item != null' &>/dev/null
}

# Validate DynamoDB data
validate_dynamodb() {
    if [[ "$SKIP_DATA" == "true" ]]; then
        log_verbose "DynamoDB validation skipped"
        return
    fi
    
    log_header "DynamoDB Data Validation"
    
    # Determine table naming patterns
    # Old: {app}-{env} or {app}-table-{env}
    # New: {app}-data-{env}-{ecosystem}
    
    local old_table_patterns=(
        "${APP_NAME}-${ENVIRONMENT}"
        "${APP_NAME}-table-${ENVIRONMENT}"
        "${APP_NAME}-data-${ENVIRONMENT}"
    )
    
    local new_table_prefix="${APP_NAME}-data-${ENVIRONMENT}"
    if [[ -n "$ECOSYSTEM" ]]; then
        new_table_prefix="${APP_NAME}-data-${ENVIRONMENT}-${ECOSYSTEM}"
    fi
    
    # List all tables and find matches
    local all_tables
    all_tables=$(aws dynamodb list-tables --query "TableNames" --output json 2>/dev/null || echo '[]')
    
    log_verbose "Looking for tables matching patterns..."
    
    # Find old table
    local old_table=""
    for pattern in "${old_table_patterns[@]}"; do
        if echo "$all_tables" | jq -e ".[] | select(. == \"$pattern\")" &>/dev/null; then
            old_table="$pattern"
            break
        fi
    done
    
    # Find new table(s)
    local new_tables
    new_tables=$(echo "$all_tables" | jq -r ".[] | select(startswith(\"$new_table_prefix\"))")
    
    if [[ -z "$old_table" ]]; then
        warn_check "Old DynamoDB table" "No old table found (may already be cleaned up)"
        
        # If no old table, check new tables exist
        if [[ -n "$new_tables" ]]; then
            while IFS= read -r new_table; do
                if [[ -n "$new_table" ]]; then
                    local count
                    count=$(get_table_count "$new_table")
                    pass_check "New table exists: $new_table" "Contains $count items"
                fi
            done <<< "$new_tables"
        fi
        return
    fi
    
    log_verbose "Old table: $old_table"
    
    if [[ -z "$new_tables" ]]; then
        fail_check "New DynamoDB tables" "No tables found matching prefix $new_table_prefix"
        return
    fi
    
    # Compare record counts
    local old_count
    old_count=$(get_table_count "$old_table")
    log_verbose "Old table count: $old_count"
    
    local total_new_count=0
    while IFS= read -r new_table; do
        if [[ -n "$new_table" ]]; then
            local count
            count=$(get_table_count "$new_table")
            log_verbose "New table $new_table count: $count"
            total_new_count=$((total_new_count + count))
        fi
    done <<< "$new_tables"
    
    if [[ "$total_new_count" -eq 0 && "$old_count" -gt 0 ]]; then
        fail_check "Data migration" "New tables are empty but old table has $old_count items"
    elif [[ "$total_new_count" -lt "$old_count" ]]; then
        local diff=$((old_count - total_new_count))
        local pct=$((100 * total_new_count / old_count))
        if [[ "$pct" -ge 95 ]]; then
            warn_check "Record count" "New tables have $total_new_count items vs old $old_count ($pct%)"
        else
            fail_check "Record count" "New tables missing $diff items (have $total_new_count of $old_count)"
        fi
    else
        pass_check "Record count" "New tables have $total_new_count items (old: $old_count)"
    fi
    
    # Spot-check data integrity
    log_step "Spot-checking $SAMPLE_SIZE records..."
    
    local sample
    sample=$(sample_records "$old_table" "$SAMPLE_SIZE")
    local sample_count
    sample_count=$(echo "$sample" | jq '.Items | length')
    
    if [[ "$sample_count" -eq 0 ]]; then
        log_verbose "No records to sample"
        return
    fi
    
    # Get table key schema
    local key_schema
    key_schema=$(aws dynamodb describe-table \
        --table-name "$old_table" \
        --query "Table.KeySchema" \
        --output json 2>/dev/null || echo '[]')
    
    local pk_name
    pk_name=$(echo "$key_schema" | jq -r '.[] | select(.KeyType == "HASH") | .AttributeName')
    local sk_name
    sk_name=$(echo "$key_schema" | jq -r '.[] | select(.KeyType == "RANGE") | .AttributeName // empty')
    
    log_verbose "Key schema: PK=$pk_name SK=$sk_name"
    
    local matched=0
    local checked=0
    
    # Use first new table for spot checks
    local check_table
    check_table=$(echo "$new_tables" | head -1)
    
    for i in $(seq 0 $((sample_count - 1))); do
        local item
        item=$(echo "$sample" | jq ".Items[$i]")
        
        # Build key JSON
        local key_json="{\"$pk_name\": $(echo "$item" | jq ".[\"$pk_name\"]")"
        if [[ -n "$sk_name" ]]; then
            key_json="$key_json, \"$sk_name\": $(echo "$item" | jq ".[\"$sk_name\"]")"
        fi
        key_json="$key_json}"
        
        ((checked++)) || true
        
        if record_exists_in_target "$check_table" "$key_json"; then
            ((matched++)) || true
        else
            log_verbose "Record not found in target: $key_json"
        fi
    done
    
    if [[ "$checked" -eq 0 ]]; then
        warn_check "Data integrity" "No records available to check"
    elif [[ "$matched" -eq "$checked" ]]; then
        pass_check "Data integrity" "All $checked sampled records found in new table"
    elif [[ "$matched" -ge $((checked * 9 / 10)) ]]; then
        warn_check "Data integrity" "$matched of $checked sampled records found ($(( 100 * matched / checked ))%)"
    else
        fail_check "Data integrity" "Only $matched of $checked sampled records found in new table"
    fi
}

# Print summary
print_summary() {
    log_header "Validation Summary"
    
    echo -e "App: ${BOLD}${APP_NAME}${NC}, Environment: ${BOLD}${ENVIRONMENT}${NC}" >&2
    [[ -n "$ECOSYSTEM" ]] && echo -e "Ecosystem: ${BOLD}${ECOSYSTEM}${NC}" >&2
    echo "" >&2
    
    echo -e "Total checks: ${BOLD}$TOTAL_CHECKS${NC}" >&2
    echo -e "  ${GREEN}Passed:${NC}   $PASSED_CHECKS" >&2
    echo -e "  ${RED}Failed:${NC}   $FAILED_CHECKS" >&2
    echo -e "  ${YELLOW}Warnings:${NC} $WARNINGS" >&2
    echo "" >&2
    
    if [[ "$FAILED_CHECKS" -eq 0 ]]; then
        echo -e "${GREEN}${BOLD}✓ VALIDATION PASSED${NC}" >&2
        if [[ "$WARNINGS" -gt 0 ]]; then
            echo -e "  (with $WARNINGS warnings - review before proceeding)" >&2
        fi
        return 0
    else
        echo -e "${RED}${BOLD}✗ VALIDATION FAILED${NC}" >&2
        echo -e "  Review failures above before proceeding with migration." >&2
        return 1
    fi
}

# Main
main() {
    parse_args "$@"
    
    echo -e "${BOLD}Migration Validation: ${APP_NAME} (${ENVIRONMENT})${NC}" >&2
    echo -e "Region: ${AWS_REGION}" >&2
    [[ -n "$ECOSYSTEM" ]] && echo -e "Ecosystem: ${ECOSYSTEM}" >&2
    echo "" >&2
    
    check_prerequisites
    
    validate_cf_outputs
    validate_dns
    validate_dynamodb
    
    print_summary
}

main "$@"
