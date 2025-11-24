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

# Test script for S3 integration using MinIO
set -e

echo "=== Testing S3 Integration with MinIO ==="

# Set environment variables
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="admin"
export AWS_SECRET_ACCESS_KEY="password"
export S3_ENDPOINT="http://localhost:9000"

# Configure AWS CLI for MinIO
aws configure set aws_access_key_id admin
aws configure set aws_secret_access_key password
aws configure set default.region us-east-1

# Create a test bucket
echo "Creating test bucket..."
aws s3 mb s3://test-bucket \
    --endpoint-url "$S3_ENDPOINT" \
    2>/dev/null || echo "Bucket already exists"

# Create a test file
echo "Creating test file..."
echo "This is a test file for S3 integration" > /tmp/test-file.txt

# Upload the test file
echo "Uploading test file..."
aws s3 cp /tmp/test-file.txt s3://test-bucket/test-file.txt \
    --endpoint-url "$S3_ENDPOINT"

# List bucket contents
echo "Listing bucket contents..."
aws s3 ls s3://test-bucket/ \
    --endpoint-url "$S3_ENDPOINT"

# Download the file
echo "Downloading test file..."
aws s3 cp s3://test-bucket/test-file.txt /tmp/downloaded-file.txt \
    --endpoint-url "$S3_ENDPOINT"

# Verify the content
echo "Verifying content..."
if diff /tmp/test-file.txt /tmp/downloaded-file.txt; then
    echo "✓ File content matches"
else
    echo "✗ File content does not match"
    exit 1
fi

# Create test YAML config
cat > /tmp/s3-test-config.yaml <<EOF
sources:
  - name: local-minio
    kind: s3
    region: us-east-1
    endpoint: http://localhost:9000
    accessKeyId: admin
    secretAccessKey: password
    forcePathStyle: true
EOF

echo ""
echo "✓ MinIO (S3) is working correctly"
echo "✓ Test bucket created: test-bucket"
echo "✓ Test config created: /tmp/s3-test-config.yaml"
echo ""
echo "MinIO Console: http://localhost:9001"
echo "  Username: admin"
echo "  Password: password"
echo ""
echo "You can now test the S3 source with:"
echo "  go test ./internal/sources/s3/... -v"
