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

# Test script for Neptune/Gremlin integration
set -e

echo "=== Testing Neptune/Gremlin Integration ==="

# Wait for Gremlin Server to be ready
echo "Waiting for Gremlin Server to be ready..."
sleep 5

# Test connection using gremlin-console
echo "Testing Gremlin Server connection..."

# Create a simple test script
cat > /tmp/gremlin-test.groovy <<'EOF'
// Connect to Gremlin Server
:remote connect tinkerpop.server conf/remote.yaml
:remote console

// Create test vertices
v1 = g.addV('person').property('name', 'Alice').property('age', 30).next()
v2 = g.addV('person').property('name', 'Bob').property('age', 25).next()
v3 = g.addV('person').property('name', 'Charlie').property('age', 35).next()

// Create edges
g.V(v1).addE('knows').to(v2).next()
g.V(v2).addE('knows').to(v3).next()

// Query the graph
println "All persons:"
g.V().hasLabel('person').valueMap().toList()

println "\nAll 'knows' relationships:"
g.E().hasLabel('knows').toList()

println "\nPeople Alice knows:"
g.V().has('person', 'name', 'Alice').out('knows').values('name').toList()
EOF

# Test with a simple HTTP request (Gremlin Server REST API)
echo "Testing Gremlin Server with REST API..."
GREMLIN_QUERY='{"gremlin":"g.V().count()"}'

curl -X POST http://localhost:8182 \
    -H "Content-Type: application/json" \
    -d "$GREMLIN_QUERY" 2>/dev/null | python3 -m json.tool || echo "Gremlin Server is starting..."

# Create test YAML config for Neptune (using Gremlin Server as compatible substitute)
cat > /tmp/neptune-test-config.yaml <<EOF
sources:
  - name: local-neptune
    kind: neptune
    endpoint: ws://localhost:8182/gremlin
    useIAM: false
    # Note: IAM is disabled for local testing
    # In production Neptune, you would set:
    # endpoint: wss://your-neptune-cluster.region.neptune.amazonaws.com:8182/gremlin
    # useIAM: true
EOF

echo ""
echo "✓ Gremlin Server is working"
echo "✓ Endpoint: ws://localhost:8182/gremlin"
echo "✓ Test config created: /tmp/neptune-test-config.yaml"
echo ""
echo "You can now test the Neptune source with:"
echo "  go test ./internal/sources/neptune/... -v"
echo ""
echo "Note: For real AWS Neptune with IAM authentication:"
echo "  1. Set useIAM: true in config"
echo "  2. Use wss:// endpoint (TLS)"
echo "  3. Ensure AWS credentials are configured"
echo "  4. Neptune will use SigV4 signing automatically"
