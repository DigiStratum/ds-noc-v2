#!/bin/bash
# cleanup-old-stacks.sh
# Safely remove old CloudFormation stacks after successful migration cutover
#
# Usage:
#   ./scripts/cleanup-old-stacks.sh <app-name> <environment>
#   ./scripts/cleanup-old-stacks.sh <app-name> <environment> --dry-run
#   ./scripts/cleanup-old-stacks.sh <app-name> <environment> --force
#
# Examples:
#   ./scripts/cleanup-old-stacks.sh dskanban prod --dry-run
#   ./scripts/cleanup-old-stacks.sh myapp dev --force
#
# Prerequisites:
#   - AWS CLI configured with appropriate permissions
#   - New stacks deployed and healthy (validates automatically)
#   - Migration validation passed (recommended to run validate-migration.sh first)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Defaults
DRY_RUN=false
FORCE=false
SKIP_VALIDATION=false
VERBOSE=false

# AWS region (default to current)
AWS_REGION="${AWS_REGION:-$(aws configure get region 2>/dev/null || echo 'us-east-1')}"

usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] <app-name> <environment>

Safely remove old CloudFormation stacks after migration cutover.

Arguments:
  app-name         Application name (e.g., 'dskanban', 'myapp')
  environment      Environment name: 'dev' or 'prod'

Options:
  --dry-run, -n    Preview what would be deleted without making changes
  --force, -f      Skip interactive confirmation (still creates backups)
  --skip-validation  Skip new stack health checks (not recommended)
  -v, --verbose    Show detailed operation info
  -h, --help       Show this help message

Safety Features:
  - Verifies new stacks are deployed and healthy before cleanup
  - Creates DynamoDB backup before any table deletion
  - Requires explicit confirmation unless --force is used
  - Deletes stacks in dependency order (app before data)
  - Cleans up orphaned resources (S3 buckets, Lambda functions)

Deletion Order:
  1. Disable DynamoDB streams (stop sync)
  2. Create final backup of old table
  3. Delete app stack (Lambda, API Gateway, CloudFront)
  4. Delete data stack (DynamoDB, S3)

Examples:
  $(basename "$0") dskanban prod --dry-run    # Preview cleanup
  $(basename "$0") myapp dev                  # Interactive cleanup
  $(basename "$0") myapp prod --force         # Non-interactive (with backups)

Stack Naming Conventions:
  Old (legacy):     {app}-{env}                    e.g., myapp-prod
  New (app):        {app}-app-{env}                e.g., myapp-app-prod
  New (data):       {app}-data-{env}-{ecosystem}   e.g., myapp-data-prod-digistratum
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

log_step() {
    echo -e "${BLUE}→${NC} $*" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "  ${BLUE}·${NC} $*" >&2
    fi
}

log_dry_run() {
    echo -e "${YELLOW}[DRY-RUN]${NC} $*" >&2
}

# Check if a CloudFormation stack exists and get its status
get_stack_status() {
    local stack_name="$1"
    aws cloudformation describe-stacks \
        --stack-name "$stack_name" \
        --region "$AWS_REGION" \
        --query 'Stacks[0].StackStatus' \
        --output text 2>/dev/null || echo "DOES_NOT_EXIST"
}

# Check if stack is in a healthy/stable state
is_stack_healthy() {
    local status="$1"
    case "$status" in
        CREATE_COMPLETE|UPDATE_COMPLETE|IMPORT_COMPLETE)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# List all old stacks matching the pattern
find_old_stacks() {
    local app_name="$1"
    local env_name="$2"
    
    # Old/legacy stack naming: {app}-{env}
    # NOT matching new patterns: {app}-app-{env} or {app}-data-{env}-*
    local old_stack="${app_name}-${env_name}"
    
    local status
    status=$(get_stack_status "$old_stack")
    
    if [[ "$status" != "DOES_NOT_EXIST" ]]; then
        echo "$old_stack"
    fi
}

# Find new stacks to validate they're healthy
find_new_stacks() {
    local app_name="$1"
    local env_name="$2"
    
    local stacks=()
    
    # New app stack: {app}-app-{env}
    local app_stack="${app_name}-app-${env_name}"
    if [[ $(get_stack_status "$app_stack") != "DOES_NOT_EXIST" ]]; then
        stacks+=("$app_stack")
    fi
    
    # New data stacks: {app}-data-{env}-{ecosystem}
    # List all stacks matching the pattern
    local data_stacks
    data_stacks=$(aws cloudformation list-stacks \
        --region "$AWS_REGION" \
        --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE IMPORT_COMPLETE \
        --query "StackSummaries[?starts_with(StackName, '${app_name}-data-${env_name}-')].StackName" \
        --output text 2>/dev/null || true)
    
    for stack in $data_stacks; do
        stacks+=("$stack")
    done
    
    echo "${stacks[@]}"
}

# Verify new stacks are healthy
verify_new_stacks_healthy() {
    local app_name="$1"
    local env_name="$2"
    
    log_step "Verifying new stacks are healthy..."
    
    local new_stacks
    new_stacks=$(find_new_stacks "$app_name" "$env_name")
    
    if [[ -z "$new_stacks" ]]; then
        log_error "No new stacks found matching ${app_name}-app-${env_name} or ${app_name}-data-${env_name}-*"
        log_error "Deploy new stacks before running cleanup"
        return 1
    fi
    
    local all_healthy=true
    for stack in $new_stacks; do
        local status
        status=$(get_stack_status "$stack")
        if is_stack_healthy "$status"; then
            log_verbose "Stack $stack: $status (healthy)"
        else
            log_error "Stack $stack: $status (unhealthy)"
            all_healthy=false
        fi
    done
    
    if [[ "$all_healthy" == "true" ]]; then
        log_info "All new stacks are healthy: $new_stacks"
        return 0
    else
        return 1
    fi
}

# Get DynamoDB table name from stack
get_table_from_stack() {
    local stack_name="$1"
    
    # Query stack resources for DynamoDB tables
    aws cloudformation list-stack-resources \
        --stack-name "$stack_name" \
        --region "$AWS_REGION" \
        --query "StackResourceSummaries[?ResourceType=='AWS::DynamoDB::Table'].PhysicalResourceId" \
        --output text 2>/dev/null || true
}

# Get S3 buckets from stack
get_buckets_from_stack() {
    local stack_name="$1"
    
    aws cloudformation list-stack-resources \
        --stack-name "$stack_name" \
        --region "$AWS_REGION" \
        --query "StackResourceSummaries[?ResourceType=='AWS::S3::Bucket'].PhysicalResourceId" \
        --output text 2>/dev/null || true
}

# Get Lambda functions from stack
get_lambdas_from_stack() {
    local stack_name="$1"
    
    aws cloudformation list-stack-resources \
        --stack-name "$stack_name" \
        --region "$AWS_REGION" \
        --query "StackResourceSummaries[?ResourceType=='AWS::Lambda::Function'].PhysicalResourceId" \
        --output text 2>/dev/null || true
}

# Disable DynamoDB streams on a table
disable_dynamodb_stream() {
    local table_name="$1"
    
    # Check if stream is enabled
    local stream_spec
    stream_spec=$(aws dynamodb describe-table \
        --table-name "$table_name" \
        --region "$AWS_REGION" \
        --query 'Table.StreamSpecification.StreamEnabled' \
        --output text 2>/dev/null || echo "false")
    
    if [[ "$stream_spec" == "true" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would disable stream on table: $table_name"
        else
            log_step "Disabling stream on table: $table_name"
            aws dynamodb update-table \
                --table-name "$table_name" \
                --region "$AWS_REGION" \
                --stream-specification StreamEnabled=false \
                --no-cli-pager >/dev/null
            log_info "Stream disabled on $table_name"
        fi
    else
        log_verbose "No stream enabled on $table_name"
    fi
}

# Create DynamoDB backup
create_dynamodb_backup() {
    local table_name="$1"
    local backup_name="pre-cleanup-$(date +%Y%m%d-%H%M%S)"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_dry_run "Would create backup: ${table_name}-${backup_name}"
        echo "DRY-RUN-BACKUP"
        return 0
    fi
    
    log_step "Creating backup of table: $table_name"
    local backup_arn
    backup_arn=$(aws dynamodb create-backup \
        --table-name "$table_name" \
        --backup-name "$backup_name" \
        --region "$AWS_REGION" \
        --query 'BackupDetails.BackupArn' \
        --output text 2>/dev/null)
    
    if [[ -n "$backup_arn" ]]; then
        log_info "Backup created: $backup_arn"
        echo "$backup_arn"
        return 0
    else
        log_error "Failed to create backup for $table_name"
        return 1
    fi
}

# Empty S3 bucket (required before deletion)
empty_s3_bucket() {
    local bucket_name="$1"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_dry_run "Would empty bucket: $bucket_name"
        return 0
    fi
    
    log_step "Emptying bucket: $bucket_name"
    
    # Delete all object versions
    aws s3api list-object-versions \
        --bucket "$bucket_name" \
        --region "$AWS_REGION" \
        --query 'Versions[].{Key:Key,VersionId:VersionId}' \
        --output json 2>/dev/null | \
        jq -c '.[]? // empty' | while read -r obj; do
            local key version_id
            key=$(echo "$obj" | jq -r '.Key')
            version_id=$(echo "$obj" | jq -r '.VersionId')
            aws s3api delete-object \
                --bucket "$bucket_name" \
                --key "$key" \
                --version-id "$version_id" \
                --region "$AWS_REGION" >/dev/null 2>&1 || true
        done
    
    # Delete delete markers
    aws s3api list-object-versions \
        --bucket "$bucket_name" \
        --region "$AWS_REGION" \
        --query 'DeleteMarkers[].{Key:Key,VersionId:VersionId}' \
        --output json 2>/dev/null | \
        jq -c '.[]? // empty' | while read -r obj; do
            local key version_id
            key=$(echo "$obj" | jq -r '.Key')
            version_id=$(echo "$obj" | jq -r '.VersionId')
            aws s3api delete-object \
                --bucket "$bucket_name" \
                --key "$key" \
                --version-id "$version_id" \
                --region "$AWS_REGION" >/dev/null 2>&1 || true
        done
    
    log_info "Bucket emptied: $bucket_name"
}

# Delete CloudFormation stack
delete_stack() {
    local stack_name="$1"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_dry_run "Would delete stack: $stack_name"
        return 0
    fi
    
    log_step "Deleting stack: $stack_name"
    
    aws cloudformation delete-stack \
        --stack-name "$stack_name" \
        --region "$AWS_REGION"
    
    # Wait for deletion to complete
    log_step "Waiting for stack deletion to complete..."
    aws cloudformation wait stack-delete-complete \
        --stack-name "$stack_name" \
        --region "$AWS_REGION" 2>/dev/null || true
    
    # Verify deletion
    local status
    status=$(get_stack_status "$stack_name")
    if [[ "$status" == "DOES_NOT_EXIST" || "$status" == "DELETE_COMPLETE" ]]; then
        log_info "Stack deleted: $stack_name"
        return 0
    else
        log_error "Stack deletion may have failed: $stack_name (status: $status)"
        return 1
    fi
}

# Find and clean up orphaned resources
cleanup_orphaned_resources() {
    local app_name="$1"
    local env_name="$2"
    
    log_step "Checking for orphaned resources..."
    
    local found_orphans=false
    
    # Check for orphaned Lambda functions
    local orphan_lambdas
    orphan_lambdas=$(aws lambda list-functions \
        --region "$AWS_REGION" \
        --query "Functions[?starts_with(FunctionName, '${app_name}-${env_name}') && !contains(FunctionName, '-app-')].FunctionName" \
        --output text 2>/dev/null || true)
    
    if [[ -n "$orphan_lambdas" ]]; then
        found_orphans=true
        for lambda_name in $orphan_lambdas; do
            if [[ "$DRY_RUN" == "true" ]]; then
                log_dry_run "Would delete orphaned Lambda: $lambda_name"
            else
                log_step "Deleting orphaned Lambda: $lambda_name"
                aws lambda delete-function \
                    --function-name "$lambda_name" \
                    --region "$AWS_REGION" 2>/dev/null || true
                log_info "Deleted Lambda: $lambda_name"
            fi
        done
    fi
    
    # Check for orphaned S3 buckets
    local orphan_buckets
    orphan_buckets=$(aws s3api list-buckets \
        --query "Buckets[?starts_with(Name, '${app_name}-${env_name}')].Name" \
        --output text 2>/dev/null || true)
    
    # Filter out buckets that belong to new stacks
    for bucket in $orphan_buckets; do
        # Skip if bucket contains 'data' pattern (belongs to new data stack)
        if [[ "$bucket" == *"-data-"* ]]; then
            log_verbose "Skipping new-stack bucket: $bucket"
            continue
        fi
        
        found_orphans=true
        if [[ "$DRY_RUN" == "true" ]]; then
            log_dry_run "Would delete orphaned S3 bucket: $bucket"
        else
            empty_s3_bucket "$bucket"
            log_step "Deleting orphaned S3 bucket: $bucket"
            aws s3api delete-bucket \
                --bucket "$bucket" \
                --region "$AWS_REGION" 2>/dev/null || true
            log_info "Deleted S3 bucket: $bucket"
        fi
    done
    
    if [[ "$found_orphans" == "false" ]]; then
        log_info "No orphaned resources found"
    fi
}

# Interactive confirmation
confirm_cleanup() {
    local old_stacks="$1"
    local tables="$2"
    local buckets="$3"
    
    echo "" >&2
    echo -e "${BOLD}=== CLEANUP SUMMARY ===${NC}" >&2
    echo "" >&2
    echo -e "${YELLOW}The following resources will be deleted:${NC}" >&2
    echo "" >&2
    
    if [[ -n "$old_stacks" ]]; then
        echo -e "${BOLD}CloudFormation Stacks:${NC}" >&2
        for stack in $old_stacks; do
            echo "  - $stack" >&2
        done
        echo "" >&2
    fi
    
    if [[ -n "$tables" ]]; then
        echo -e "${BOLD}DynamoDB Tables (backup will be created):${NC}" >&2
        for table in $tables; do
            echo "  - $table" >&2
        done
        echo "" >&2
    fi
    
    if [[ -n "$buckets" ]]; then
        echo -e "${BOLD}S3 Buckets (will be emptied and deleted):${NC}" >&2
        for bucket in $buckets; do
            echo "  - $bucket" >&2
        done
        echo "" >&2
    fi
    
    echo -e "${RED}${BOLD}WARNING: This action is IRREVERSIBLE (except for DynamoDB backups).${NC}" >&2
    echo "" >&2
    
    read -p "Type 'DELETE' to confirm: " confirmation
    if [[ "$confirmation" == "DELETE" ]]; then
        return 0
    else
        log_warn "Cleanup cancelled"
        return 1
    fi
}

# Generate cleanup report
generate_report() {
    local app_name="$1"
    local env_name="$2"
    local deleted_stacks="$3"
    local backups_created="$4"
    local deleted_buckets="$5"
    
    echo "" >&2
    echo -e "${BOLD}=== CLEANUP REPORT ===${NC}" >&2
    echo "" >&2
    
    if [[ -n "$deleted_stacks" ]]; then
        echo -e "${GREEN}Deleted Stacks:${NC}" >&2
        for stack in $deleted_stacks; do
            echo "  ✓ $stack" >&2
        done
        echo "" >&2
    fi
    
    if [[ -n "$backups_created" ]]; then
        echo -e "${GREEN}DynamoDB Backups Created:${NC}" >&2
        for backup in $backups_created; do
            echo "  ✓ $backup" >&2
        done
        echo "" >&2
    fi
    
    if [[ -n "$deleted_buckets" ]]; then
        echo -e "${GREEN}Deleted S3 Buckets:${NC}" >&2
        for bucket in $deleted_buckets; do
            echo "  ✓ $bucket" >&2
        done
        echo "" >&2
    fi
    
    log_info "Cleanup complete for ${app_name}-${env_name}"
}

# Main cleanup flow
main() {
    local app_name="$1"
    local env_name="$2"
    
    echo "" >&2
    echo -e "${BOLD}Old Stack Cleanup Script${NC}" >&2
    echo -e "App: ${app_name}, Environment: ${env_name}" >&2
    echo -e "Region: ${AWS_REGION}" >&2
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}Mode: DRY-RUN (no changes will be made)${NC}" >&2
    fi
    echo "" >&2
    
    # Step 1: Verify new stacks are healthy
    if [[ "$SKIP_VALIDATION" != "true" ]]; then
        if ! verify_new_stacks_healthy "$app_name" "$env_name"; then
            log_error "New stacks are not healthy. Aborting cleanup."
            log_error "Fix new stacks or use --skip-validation (not recommended)"
            exit 1
        fi
    else
        log_warn "Skipping new stack validation (--skip-validation)"
    fi
    
    # Step 2: Find old stacks to delete
    local old_stacks
    old_stacks=$(find_old_stacks "$app_name" "$env_name")
    
    if [[ -z "$old_stacks" ]]; then
        log_info "No old stacks found matching ${app_name}-${env_name}"
        log_info "Looking for orphaned resources..."
        cleanup_orphaned_resources "$app_name" "$env_name"
        exit 0
    fi
    
    log_info "Found old stacks: $old_stacks"
    
    # Step 3: Discover resources in old stacks
    local all_tables=""
    local all_buckets=""
    
    for stack in $old_stacks; do
        local tables buckets
        tables=$(get_table_from_stack "$stack")
        buckets=$(get_buckets_from_stack "$stack")
        
        if [[ -n "$tables" ]]; then
            all_tables="$all_tables $tables"
            log_verbose "Found tables in $stack: $tables"
        fi
        if [[ -n "$buckets" ]]; then
            all_buckets="$all_buckets $buckets"
            log_verbose "Found buckets in $stack: $buckets"
        fi
    done
    
    all_tables=$(echo "$all_tables" | xargs)  # Trim whitespace
    all_buckets=$(echo "$all_buckets" | xargs)
    
    # Step 4: Confirm deletion (unless --force or --dry-run)
    if [[ "$DRY_RUN" != "true" && "$FORCE" != "true" ]]; then
        if ! confirm_cleanup "$old_stacks" "$all_tables" "$all_buckets"; then
            exit 0
        fi
    fi
    
    # Track what we delete for the report
    local deleted_stacks=""
    local backups_created=""
    local deleted_buckets=""
    
    # Step 5: Disable DynamoDB streams (stop sync before backup)
    for table in $all_tables; do
        disable_dynamodb_stream "$table"
    done
    
    # Step 6: Create DynamoDB backups
    for table in $all_tables; do
        local backup_arn
        if backup_arn=$(create_dynamodb_backup "$table"); then
            backups_created="$backups_created $backup_arn"
        else
            log_error "Failed to create backup for $table. Aborting."
            exit 1
        fi
    done
    
    if [[ "$DRY_RUN" != "true" && -n "$backups_created" ]]; then
        log_info "All backups created successfully"
    fi
    
    # Step 7: Empty S3 buckets (required before stack deletion)
    for bucket in $all_buckets; do
        empty_s3_bucket "$bucket"
        deleted_buckets="$deleted_buckets $bucket"
    done
    
    # Step 8: Delete stacks (app stack before data, but in legacy mode it's usually one stack)
    for stack in $old_stacks; do
        if delete_stack "$stack"; then
            deleted_stacks="$deleted_stacks $stack"
        fi
    done
    
    # Step 9: Clean up orphaned resources
    cleanup_orphaned_resources "$app_name" "$env_name"
    
    # Step 10: Generate report
    if [[ "$DRY_RUN" != "true" ]]; then
        generate_report "$app_name" "$env_name" "$deleted_stacks" "$backups_created" "$deleted_buckets"
    else
        echo "" >&2
        log_info "Dry run complete. No changes were made."
    fi
}

# Parse arguments
APP_NAME=""
ENV_NAME=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        --skip-validation)
            SKIP_VALIDATION=true
            shift
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
            if [[ -z "$APP_NAME" ]]; then
                APP_NAME="$1"
            elif [[ -z "$ENV_NAME" ]]; then
                ENV_NAME="$1"
            else
                log_error "Unexpected argument: $1"
                usage
            fi
            shift
            ;;
    esac
done

# Validate arguments
if [[ -z "$APP_NAME" ]]; then
    log_error "App name is required"
    echo "" >&2
    usage
fi

if [[ -z "$ENV_NAME" ]]; then
    log_error "Environment is required"
    echo "" >&2
    usage
fi

if [[ "$ENV_NAME" != "dev" && "$ENV_NAME" != "prod" ]]; then
    log_error "Environment must be 'dev' or 'prod', got: $ENV_NAME"
    exit 1
fi

# Verify AWS CLI is available
if ! command -v aws &>/dev/null; then
    log_error "AWS CLI is required but not installed"
    exit 1
fi

# Verify AWS credentials
if ! aws sts get-caller-identity &>/dev/null; then
    log_error "AWS credentials not configured or expired"
    exit 1
fi

# Run main
main "$APP_NAME" "$ENV_NAME"
