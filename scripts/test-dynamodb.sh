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

# Test script for DynamoDB integration using DynamoDB Local
set -e

echo "=== Testing DynamoDB Integration ==="

# Set environment variables
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export DYNAMODB_ENDPOINT="http://localhost:8000"

# Create a test table
echo "Creating test table..."
aws dynamodb create-table \
    --table-name TestTable \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --endpoint-url "$DYNAMODB_ENDPOINT" \
    --region "$AWS_REGION" \
    2>/dev/null || echo "Table already exists"

# Wait for table to be active
echo "Waiting for table to be ready..."
sleep 2

# Insert test data
echo "Inserting test data..."
aws dynamodb put-item \
    --table-name TestTable \
    --item '{
        "id": {"S": "test-1"},
        "name": {"S": "Test Item 1"},
        "value": {"N": "100"}
    }' \
    --endpoint-url "$DYNAMODB_ENDPOINT" \
    --region "$AWS_REGION"

aws dynamodb put-item \
    --table-name TestTable \
    --item '{
        "id": {"S": "test-2"},
        "name": {"S": "Test Item 2"},
        "value": {"N": "200"}
    }' \
    --endpoint-url "$DYNAMODB_ENDPOINT" \
    --region "$AWS_REGION"

# Query the data
echo "Querying test data..."
aws dynamodb scan \
    --table-name TestTable \
    --endpoint-url "$DYNAMODB_ENDPOINT" \
    --region "$AWS_REGION"

# Create test YAML config
cat > /tmp/dynamodb-test-config.yaml <<EOF
sources:
  - name: local-dynamodb
    kind: dynamodb
    region: us-east-1
    endpoint: http://localhost:8000
    accessKeyId: test
    secretAccessKey: test
EOF

echo ""
echo "✓ DynamoDB Local is working correctly"
echo "✓ Test table created: TestTable"
echo "✓ Test config created: /tmp/dynamodb-test-config.yaml"
echo ""
echo "You can now test the DynamoDB source with:"
echo "  go test ./internal/sources/dynamodb/... -v"
