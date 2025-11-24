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

# Test script for MongoDB/DocumentDB integration
set -e

echo "=== Testing MongoDB/DocumentDB Integration ==="

# Set environment variables
export MONGODB_URI="mongodb://admin:password@localhost:27017"

# Test connection and insert data
echo "Testing MongoDB connection..."
docker exec genai-toolbox-mongodb mongosh --quiet "$MONGODB_URI" --eval "
    // Switch to test database
    db = db.getSiblingDB('testdb');

    // Create test collection and insert data
    db.users.insertMany([
        { username: 'alice', email: 'alice@example.com', age: 30 },
        { username: 'bob', email: 'bob@example.com', age: 25 },
        { username: 'charlie', email: 'charlie@example.com', age: 35 }
    ]);

    print('Inserted test data');

    // Query the data
    print('\\nUsers in database:');
    db.users.find().forEach(printjson);
"

# Test aggregation (DocumentDB-compatible)
echo ""
echo "Testing aggregation pipeline..."
docker exec genai-toolbox-mongodb mongosh --quiet "$MONGODB_URI/testdb" --eval "
    db.users.aggregate([
        { \$group: { _id: null, avgAge: { \$avg: '\$age' } } }
    ]).forEach(printjson);
"

# Create test YAML config for DocumentDB (using MongoDB as compatible substitute)
cat > /tmp/documentdb-test-config.yaml <<EOF
sources:
  - name: local-documentdb
    kind: documentdb
    uri: mongodb://admin:password@localhost:27017/testdb?directConnection=true
    # Note: TLS is disabled for local testing
    # In production DocumentDB, you would specify:
    # tlsCAFile: /path/to/rds-combined-ca-bundle.pem
EOF

echo ""
echo "✓ MongoDB is working correctly"
echo "✓ Test database: testdb"
echo "✓ Test collection: users"
echo "✓ Test config created: /tmp/documentdb-test-config.yaml"
echo ""
echo "You can now test the DocumentDB source with:"
echo "  go test ./internal/sources/documentdb/... -v"
echo ""
echo "Note: For real AWS DocumentDB, you'll need:"
echo "  1. Download CA bundle: wget https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"
echo "  2. Add tlsCAFile parameter to config"
echo "  3. Use DocumentDB cluster endpoint"
