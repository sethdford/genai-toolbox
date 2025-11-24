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

# Test script for PostgreSQL/Redshift integration
set -e

echo "=== Testing PostgreSQL/Redshift Integration ==="

# Set environment variables
export PGHOST="localhost"
export PGPORT="5432"
export PGUSER="postgres"
export PGPASSWORD="postgres"
export PGDATABASE="testdb"

# Test connection
echo "Testing database connection..."
psql -c "SELECT version();"

# Query test data
echo "Querying test data..."
psql -c "SELECT * FROM testschema.users;"
psql -c "SELECT * FROM testschema.products;"

# Run a JOIN query (Redshift-style)
echo "Running JOIN query..."
psql -c "
    SELECT
        u.username,
        u.email,
        COUNT(*) as product_count
    FROM testschema.users u
    CROSS JOIN testschema.products p
    GROUP BY u.username, u.email
    ORDER BY u.username;
"

# Create test YAML config for Postgres
cat > /tmp/postgres-test-config.yaml <<EOF
sources:
  - name: local-postgres
    kind: postgres
    host: localhost
    port: "5432"
    user: postgres
    password: postgres
    database: testdb
    queryParams:
      sslmode: disable
      application_name: genai-toolbox-test
EOF

# Create test YAML config for Redshift (using Postgres as compatible substitute)
cat > /tmp/redshift-test-config.yaml <<EOF
sources:
  - name: local-redshift
    kind: redshift
    host: localhost
    port: "5432"
    user: postgres
    password: postgres
    database: testdb
    queryParams:
      sslmode: disable
EOF

echo ""
echo "✓ PostgreSQL is working correctly"
echo "✓ Test schema created: testschema"
echo "✓ Test tables: users, products"
echo "✓ Test config created: /tmp/postgres-test-config.yaml"
echo "✓ Test config created: /tmp/redshift-test-config.yaml"
echo ""
echo "You can now test the sources with:"
echo "  go test ./internal/sources/postgres/... -v"
echo "  go test ./internal/sources/redshift/... -v"
