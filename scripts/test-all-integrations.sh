#!/bin/bash
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Comprehensive integration test suite
# Tests all AWS integrations against local emulators and real AWS services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}=== GenAI Toolbox - Comprehensive Integration Tests ===${NC}\n"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -e "${BLUE}Running: $test_name${NC}"

    if eval "$test_command"; then
        echo -e "${GREEN}✓ PASSED: $test_name${NC}\n"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAILED: $test_name${NC}\n"
        ((TESTS_FAILED++))
    fi
}

skip_test() {
    local test_name="$1"
    local reason="$2"

    echo -e "${YELLOW}⊘ SKIPPED: $test_name${NC}"
    echo -e "${YELLOW}  Reason: $reason${NC}\n"
    ((TESTS_SKIPPED++))
}

# Test 1: Unit Tests
echo -e "${BLUE}=== Phase 1: Unit Tests ===${NC}\n"

run_test "DynamoDB Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/dynamodb/... -v -short"
run_test "S3 Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/s3/... -v -short"
run_test "Redshift Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/redshift/... -v -short"
run_test "DocumentDB Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/documentdb/... -v -short"
run_test "Neptune Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/neptune/... -v -short"
run_test "Timestream Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/timestream/... -v -short"
run_test "QLDB Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/qldb/... -v -short"
run_test "Athena Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/athena/... -v -short"
run_test "Postgres Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/postgres/... -v -short"
run_test "Tableau Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/tableau/... -v -short"
run_test "Honeycomb Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/honeycomb/... -v -short"
run_test "Splunk Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/splunk/... -v -short"
run_test "CloudWatch Unit Tests" "cd $PROJECT_ROOT && go test ./internal/sources/cloudwatch/... -v -short"

# Test 2: Local Integration Tests (requires Docker services)
echo -e "${BLUE}=== Phase 2: Local Integration Tests ===${NC}\n"

if docker info > /dev/null 2>&1; then
    echo "Starting local services..."
    $SCRIPT_DIR/validate-local.sh

    run_test "DynamoDB Integration" "$SCRIPT_DIR/test-dynamodb.sh"
    run_test "S3/MinIO Integration" "$SCRIPT_DIR/test-s3.sh"
    run_test "PostgreSQL Integration" "$SCRIPT_DIR/test-postgres.sh"
    run_test "MongoDB/DocumentDB Integration" "$SCRIPT_DIR/test-mongodb.sh"
    run_test "Gremlin/Neptune Integration" "$SCRIPT_DIR/test-neptune.sh"
else
    skip_test "Local Integration Tests" "Docker is not running"
fi

# Test 3: Real AWS Integration Tests (optional)
echo -e "${BLUE}=== Phase 3: Real AWS Integration Tests ===${NC}\n"

if [ -n "$AWS_REGION" ] && [ -n "$RUN_AWS_TESTS" ]; then
    echo -e "${YELLOW}Testing against real AWS services in region: $AWS_REGION${NC}\n"

    # Only run if explicit flag is set
    if [ "$RUN_AWS_TESTS" = "true" ]; then
        run_test "AWS DynamoDB Integration" "cd $PROJECT_ROOT && go test ./internal/sources/dynamodb/... -v -tags=integration"
        run_test "AWS S3 Integration" "cd $PROJECT_ROOT && go test ./internal/sources/s3/... -v -tags=integration"
        run_test "AWS Athena Integration" "cd $PROJECT_ROOT && go test ./internal/sources/athena/... -v -tags=integration"
        run_test "AWS Timestream Integration" "cd $PROJECT_ROOT && go test ./internal/sources/timestream/... -v -tags=integration"
        run_test "AWS QLDB Integration" "cd $PROJECT_ROOT && go test ./internal/sources/qldb/... -v -tags=integration"
        run_test "AWS CloudWatch Integration" "cd $PROJECT_ROOT && go test ./internal/sources/cloudwatch/... -v -tags=integration"

        # These require specific cluster endpoints
        if [ -n "$REDSHIFT_ENDPOINT" ]; then
            run_test "AWS Redshift Integration" "cd $PROJECT_ROOT && go test ./internal/sources/redshift/... -v -tags=integration"
        else
            skip_test "AWS Redshift Integration" "REDSHIFT_ENDPOINT not set"
        fi

        if [ -n "$DOCUMENTDB_ENDPOINT" ]; then
            run_test "AWS DocumentDB Integration" "cd $PROJECT_ROOT && go test ./internal/sources/documentdb/... -v -tags=integration"
        else
            skip_test "AWS DocumentDB Integration" "DOCUMENTDB_ENDPOINT not set"
        fi

        if [ -n "$NEPTUNE_ENDPOINT" ]; then
            run_test "AWS Neptune Integration" "cd $PROJECT_ROOT && go test ./internal/sources/neptune/... -v -tags=integration"
        else
            skip_test "AWS Neptune Integration" "NEPTUNE_ENDPOINT not set"
        fi
    else
        skip_test "AWS Integration Tests" "RUN_AWS_TESTS not set to 'true'"
    fi
else
    skip_test "Real AWS Integration Tests" "AWS_REGION not set or RUN_AWS_TESTS not enabled"
fi

# Test 4: Test Coverage
echo -e "${BLUE}=== Phase 4: Test Coverage Analysis ===${NC}\n"

run_test "Coverage Analysis" "cd $PROJECT_ROOT && go test ./internal/sources/... -coverprofile=/tmp/coverage.out && go tool cover -func=/tmp/coverage.out"

# Summary
echo -e "${BLUE}=== Test Summary ===${NC}\n"
echo -e "Tests Passed:  ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed:  ${RED}$TESTS_FAILED${NC}"
echo -e "Tests Skipped: ${YELLOW}$TESTS_SKIPPED${NC}"
echo -e "Total Tests:   $((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed${NC}"
    exit 1
fi
