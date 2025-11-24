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

# Local validation script for genai-toolbox AWS integrations
# This script validates implementations against local AWS service emulators

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOCKER_COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.local.yml"

echo -e "${BLUE}=== GenAI Toolbox Local Validation ===${NC}\n"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}WARNING: docker-compose not found. Trying 'docker compose' instead.${NC}"
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo -e "${GREEN}✓${NC} Docker is running"

# Start local services
echo -e "\n${BLUE}Starting local AWS service emulators...${NC}"
$DOCKER_COMPOSE -f "$DOCKER_COMPOSE_FILE" up -d

# Wait for services to be ready
echo -e "${YELLOW}Waiting for services to start...${NC}"
sleep 5

# Check DynamoDB Local
echo -e "\n${BLUE}Validating DynamoDB Local...${NC}"
if curl -s http://localhost:8000 > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} DynamoDB Local is running on http://localhost:8000"
else
    echo -e "${RED}✗${NC} DynamoDB Local is not responding"
fi

# Check LocalStack (S3, Athena, etc.)
echo -e "\n${BLUE}Validating LocalStack...${NC}"
if curl -s http://localhost:4566/_localstack/health | grep -q "running"; then
    echo -e "${GREEN}✓${NC} LocalStack is running on http://localhost:4566"
else
    echo -e "${RED}✗${NC} LocalStack is not responding"
fi

# Check MinIO (S3-compatible)
echo -e "\n${BLUE}Validating MinIO...${NC}"
if curl -s http://localhost:9000/minio/health/live > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} MinIO is running on http://localhost:9000"
else
    echo -e "${RED}✗${NC} MinIO is not responding"
fi

# Check PostgreSQL (for Redshift-compatible testing)
echo -e "\n${BLUE}Validating PostgreSQL...${NC}"
if docker exec genai-toolbox-postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} PostgreSQL is running on localhost:5432"
else
    echo -e "${RED}✗${NC} PostgreSQL is not responding"
fi

# Check MongoDB (for DocumentDB-compatible testing)
echo -e "\n${BLUE}Validating MongoDB...${NC}"
if docker exec genai-toolbox-mongodb mongosh --quiet --eval "db.adminCommand('ping')" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} MongoDB is running on localhost:27017"
else
    echo -e "${RED}✗${NC} MongoDB is not responding"
fi

# Run Go tests with local endpoints
echo -e "\n${BLUE}Running Go tests against local services...${NC}"

# Set environment variables for local testing
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export DYNAMODB_ENDPOINT="http://localhost:8000"
export S3_ENDPOINT="http://localhost:9000"
export LOCALSTACK_ENDPOINT="http://localhost:4566"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="postgres"
export POSTGRES_DB="testdb"
export MONGODB_URI="mongodb://localhost:27017"

cd "$PROJECT_ROOT"

# Run unit tests
echo -e "\n${YELLOW}Running unit tests...${NC}"
go test ./internal/sources/dynamodb/... -v || echo -e "${RED}DynamoDB tests failed${NC}"
go test ./internal/sources/s3/... -v || echo -e "${RED}S3 tests failed${NC}"
go test ./internal/sources/redshift/... -v || echo -e "${RED}Redshift tests failed${NC}"
go test ./internal/sources/documentdb/... -v || echo -e "${RED}DocumentDB tests failed${NC}"

echo -e "\n${BLUE}=== Validation Summary ===${NC}"
echo -e "Local services are running. You can now:"
echo -e "  1. Run integration tests: ${GREEN}go test ./...${NC}"
echo -e "  2. Test individual sources manually"
echo -e "  3. View service UIs:"
echo -e "     - MinIO Console: ${YELLOW}http://localhost:9001${NC} (admin/password)"
echo -e "     - LocalStack Dashboard: ${YELLOW}http://localhost:4566${NC}"
echo -e "\nTo stop services: ${YELLOW}$DOCKER_COMPOSE -f $DOCKER_COMPOSE_FILE down${NC}"
