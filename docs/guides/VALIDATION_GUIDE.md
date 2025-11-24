# Local Validation Guide for GenAI Toolbox

This guide walks you through validating all AWS database integrations locally before deploying to production.

## Overview

The GenAI Toolbox now includes comprehensive local validation infrastructure:
- **Docker Compose** setup with all necessary service emulators
- **Individual test scripts** for each integration
- **Comprehensive test suite** that runs everything
- **Real AWS testing** capabilities for final validation

## Quick Start (5 minutes)

```bash
# 1. Start all local services
./scripts/validate-local.sh

# 2. Run all tests
./scripts/test-all-integrations.sh

# 3. View results
# All tests should pass ✓
```

## Detailed Validation Steps

### Step 1: Start Local Services

```bash
./scripts/validate-local.sh
```

This starts:
- **DynamoDB Local** (port 8000)
- **MinIO** (S3-compatible, ports 9000/9001)
- **LocalStack** (multi-service AWS emulator, port 4566)
- **PostgreSQL** (Redshift-compatible, port 5432)
- **MongoDB** (DocumentDB-compatible, port 27017)
- **Gremlin Server** (Neptune-compatible, port 8182)

### Step 2: Test Individual Integrations

#### DynamoDB
```bash
./scripts/test-dynamodb.sh
```
- Creates test table
- Inserts sample data
- Validates queries work
- Generates test config at `/tmp/dynamodb-test-config.yaml`

**Expected Output:**
```
✓ DynamoDB Local is working correctly
✓ Test table created: TestTable
✓ Test config created: /tmp/dynamodb-test-config.yaml
```

#### S3 (MinIO)
```bash
./scripts/test-s3.sh
```
- Creates test bucket
- Uploads/downloads files
- Verifies content integrity
- Generates test config at `/tmp/s3-test-config.yaml`

**Expected Output:**
```
✓ MinIO (S3) is working correctly
✓ Test bucket created: test-bucket
MinIO Console: http://localhost:9001 (admin/password)
```

#### PostgreSQL/Redshift
```bash
./scripts/test-postgres.sh
```
- Tests database connection
- Runs JOIN queries
- Validates Redshift-compatible SQL
- Generates test configs

**Expected Output:**
```
✓ PostgreSQL is working correctly
✓ Test schema created: testschema
✓ Test tables: users, products
```

#### MongoDB/DocumentDB
```bash
./scripts/test-mongodb.sh
```
- Inserts test documents
- Runs aggregation pipelines
- Tests DocumentDB-compatible operations

**Expected Output:**
```
✓ MongoDB is working correctly
✓ Test database: testdb
✓ Test collection: users
```

#### Gremlin/Neptune
```bash
./scripts/test-neptune.sh
```
- Tests Gremlin Server connection
- Creates graph vertices/edges
- Validates graph traversals

**Expected Output:**
```
✓ Gremlin Server is working
✓ Endpoint: ws://localhost:8182/gremlin
```

### Step 3: Run Go Unit Tests

```bash
cd /Users/sethford/Documents/workspace/genai-toolbox

# Test all sources
go test ./internal/sources/... -v

# Test specific source
go test ./internal/sources/dynamodb/... -v
go test ./internal/sources/s3/... -v
go test ./internal/sources/neptune/... -v
```

### Step 4: Run Comprehensive Test Suite

```bash
./scripts/test-all-integrations.sh
```

This runs:
1. **Unit tests** for all 13 integrations
2. **Local integration tests** against Docker services
3. **Coverage analysis** to ensure quality

**Expected Output:**
```
=== Test Summary ===
Tests Passed:  25
Tests Failed:  0
Tests Skipped: 3
✓ All tests passed!
```

### Step 5: Test Against Real AWS Services (Optional)

```bash
# Set your AWS region
export AWS_REGION="us-east-1"

# Enable AWS tests
export RUN_AWS_TESTS="true"

# For cluster-based services, provide endpoints
export REDSHIFT_ENDPOINT="your-cluster.region.redshift.amazonaws.com:5439"
export DOCUMENTDB_ENDPOINT="your-cluster.region.docdb.amazonaws.com:27017"
export NEPTUNE_ENDPOINT="wss://your-cluster.region.neptune.amazonaws.com:8182/gremlin"

# Run tests
./scripts/test-all-integrations.sh
```

## Service Access

### MinIO Console (S3)
- **URL:** http://localhost:9001
- **Username:** admin
- **Password:** password

### LocalStack Dashboard
- **URL:** http://localhost:4566
- **Health:** http://localhost:4566/_localstack/health

### DynamoDB Local
- **Endpoint:** http://localhost:8000
- **AWS CLI:** `aws dynamodb list-tables --endpoint-url http://localhost:8000`

### PostgreSQL
- **Host:** localhost:5432
- **Database:** testdb
- **User:** postgres
- **Password:** postgres
- **Connect:** `psql -h localhost -U postgres -d testdb`

### MongoDB
- **URI:** mongodb://admin:password@localhost:27017
- **Connect:** `docker exec -it genai-toolbox-mongodb mongosh`

### Gremlin Server
- **Endpoint:** ws://localhost:8182/gremlin
- **REST API:** http://localhost:8182

## Test Configuration Files

After running test scripts, YAML configs are created in `/tmp/`:

```yaml
# Example: /tmp/dynamodb-test-config.yaml
sources:
  - name: local-dynamodb
    kind: dynamodb
    region: us-east-1
    endpoint: http://localhost:8000
    accessKeyId: test
    secretAccessKey: test
```

Use these as templates for your production configurations.

## Validation Checklist

- [ ] All local services start successfully
- [ ] DynamoDB test creates table and inserts data
- [ ] S3 test uploads/downloads files correctly
- [ ] PostgreSQL test runs queries successfully
- [ ] MongoDB test runs aggregations correctly
- [ ] Gremlin Server accepts connections
- [ ] All Go unit tests pass
- [ ] Test coverage is above 70%
- [ ] No security vulnerabilities in code
- [ ] All Close() methods implemented
- [ ] Real AWS services tested (if available)

## Common Issues and Solutions

### Issue: Port Already in Use

```bash
# Find what's using the port
lsof -i :8000

# Stop conflicting services
docker-compose -f scripts/docker-compose.local.yml down
```

### Issue: Services Not Starting

```bash
# Check Docker logs
docker-compose -f scripts/docker-compose.local.yml logs

# Restart specific service
docker-compose -f scripts/docker-compose.local.yml restart dynamodb-local
```

### Issue: Connection Refused

```bash
# Wait longer for services to start
sleep 10

# Check service health
docker ps
curl http://localhost:8000  # DynamoDB
curl http://localhost:9000  # MinIO
```

### Issue: Test Failures

```bash
# Run with verbose output
go test ./internal/sources/dynamodb/... -v -count=1

# Check environment variables
env | grep AWS_
env | grep DYNAMODB_

# Verify Docker services are running
docker ps | grep genai-toolbox
```

## Clean Up

### Stop Services
```bash
docker-compose -f scripts/docker-compose.local.yml down
```

### Remove All Data
```bash
docker-compose -f scripts/docker-compose.local.yml down -v
```

### Remove Test Configs
```bash
rm /tmp/*-test-config.yaml
rm /tmp/test-file.txt
rm /tmp/coverage.out
```

## Performance Testing

### DynamoDB Throughput
```bash
# Test write throughput
for i in {1..1000}; do
  aws dynamodb put-item \
    --table-name TestTable \
    --item "{\"id\": {\"S\": \"test-$i\"}}" \
    --endpoint-url http://localhost:8000 &
done
wait
```

### S3 Upload Speed
```bash
# Create large test file
dd if=/dev/zero of=/tmp/large-file.bin bs=1M count=100

# Time upload
time aws s3 cp /tmp/large-file.bin s3://test-bucket/ \
  --endpoint-url http://localhost:9000
```

### PostgreSQL Query Performance
```bash
psql -h localhost -U postgres -d testdb << EOF
EXPLAIN ANALYZE
SELECT u.*, COUNT(p.*)
FROM testschema.users u
CROSS JOIN testschema.products p
GROUP BY u.id;
EOF
```

## Next Steps

After local validation:

1. **Review test coverage:** `go tool cover -html=/tmp/coverage.out`
2. **Test against real AWS:** Set `RUN_AWS_TESTS=true`
3. **Security audit:** Review IAM permissions and credentials handling
4. **Performance testing:** Load test with production-like data
5. **Documentation:** Review package docs and examples
6. **Deployment:** Follow production deployment guide

## Resources

- **Scripts Directory:** `/Users/sethford/Documents/workspace/genai-toolbox/scripts/`
- **Docker Compose:** `scripts/docker-compose.local.yml`
- **Test Configs:** `/tmp/*-test-config.yaml`
- **Coverage Report:** `/tmp/coverage.out`

## Support

For issues:
- Check `scripts/README.md` for detailed documentation
- Review Docker logs: `docker-compose logs [service]`
- Run tests with `-v` flag for verbose output
- Check AWS credentials: `aws sts get-caller-identity`

## Production Readiness

Once all validations pass:
- ✅ Local emulator tests pass
- ✅ Unit test coverage > 70%
- ✅ Integration tests pass
- ✅ No security vulnerabilities
- ✅ Real AWS tests pass (optional)
- ✅ Performance is acceptable
- ✅ Documentation is complete

You're ready to deploy to production! See `PRODUCTION_DEPLOYMENT_GUIDE.md` (coming soon).
