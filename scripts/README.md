# Local Validation Scripts for GenAI Toolbox

This directory contains scripts for validating AWS database integrations locally using Docker containers and service emulators.

## Quick Start

1. **Start all local services:**
   ```bash
   chmod +x scripts/*.sh
   ./scripts/validate-local.sh
   ```

2. **Test individual integrations:**
   ```bash
   ./scripts/test-dynamodb.sh    # DynamoDB Local
   ./scripts/test-s3.sh          # MinIO (S3-compatible)
   ./scripts/test-postgres.sh    # PostgreSQL (Redshift-compatible)
   ./scripts/test-mongodb.sh     # MongoDB (DocumentDB-compatible)
   ./scripts/test-neptune.sh     # Gremlin Server (Neptune-compatible)
   ```

3. **Run Go tests:**
   ```bash
   go test ./internal/sources/... -v
   ```

4. **Stop all services:**
   ```bash
   docker-compose -f scripts/docker-compose.local.yml down
   ```

## Services Included

### DynamoDB Local
- **Port:** 8000
- **Purpose:** Test DynamoDB integration without AWS account
- **Test script:** `test-dynamodb.sh`

### MinIO (S3-compatible)
- **API Port:** 9000
- **Console Port:** 9001
- **Credentials:** admin/password
- **Purpose:** Test S3 integration locally
- **Test script:** `test-s3.sh`
- **Console:** http://localhost:9001

### LocalStack
- **Port:** 4566
- **Services:** S3, Athena, Glue, DynamoDB, STS, IAM
- **Purpose:** Multi-service AWS emulator
- **Credentials:** test/test

### PostgreSQL
- **Port:** 5432
- **Database:** testdb
- **Credentials:** postgres/postgres
- **Purpose:** Test Postgres and Redshift integrations
- **Test script:** `test-postgres.sh`

### MongoDB
- **Port:** 27017
- **Credentials:** admin/password
- **Purpose:** Test DocumentDB integration (MongoDB-compatible)
- **Test script:** `test-mongodb.sh`

### Gremlin Server
- **Port:** 8182
- **Purpose:** Test Neptune integration (Gremlin-compatible)
- **Test script:** `test-neptune.sh`

## Prerequisites

- Docker and Docker Compose installed
- AWS CLI installed (for testing DynamoDB and S3)
- Go 1.21+ installed
- PostgreSQL client (psql) for testing Postgres/Redshift
- Bash shell

## Directory Structure

```
scripts/
├── README.md                    # This file
├── validate-local.sh           # Main validation script
├── docker-compose.local.yml    # Docker Compose configuration
├── postgres-init.sql           # PostgreSQL initialization
├── gremlin-server.yaml         # Gremlin Server configuration
├── test-dynamodb.sh            # DynamoDB testing
├── test-s3.sh                  # S3/MinIO testing
├── test-postgres.sh            # PostgreSQL/Redshift testing
├── test-mongodb.sh             # MongoDB/DocumentDB testing
└── test-neptune.sh             # Neptune/Gremlin testing
```

## Testing Real AWS Services

While these scripts test against local emulators, you should also validate against real AWS services before production deployment:

### DynamoDB
```bash
export AWS_REGION="us-east-1"
# Use real AWS credentials
go test ./internal/sources/dynamodb/... -v
```

### S3
```bash
export AWS_REGION="us-east-1"
# Use real AWS credentials
go test ./internal/sources/s3/... -v
```

### Redshift
- Requires a real Redshift cluster
- Update config with cluster endpoint
- Ensure security group allows your IP

### DocumentDB
- Requires a real DocumentDB cluster
- Download CA bundle: https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem
- Update config with cluster endpoint and TLS settings

### Neptune
- Requires a real Neptune cluster
- Set `useIAM: true` for IAM authentication
- Use wss:// endpoint
- Ensure AWS credentials are configured

### Athena
- Requires S3 bucket for query results
- Update config with workgroup and output location

### Timestream
- Requires real Timestream database
- Update config with database and table names

### QLDB
- Requires real QLDB ledger
- Update config with ledger name

## Configuration Examples

After running the test scripts, YAML configuration examples are created in `/tmp/`:
- `/tmp/dynamodb-test-config.yaml`
- `/tmp/s3-test-config.yaml`
- `/tmp/postgres-test-config.yaml`
- `/tmp/redshift-test-config.yaml`
- `/tmp/documentdb-test-config.yaml`
- `/tmp/neptune-test-config.yaml`

You can use these as templates for your production configurations.

## Troubleshooting

### Services not starting
```bash
# Check Docker logs
docker-compose -f scripts/docker-compose.local.yml logs

# Restart services
docker-compose -f scripts/docker-compose.local.yml restart
```

### Port already in use
```bash
# Check what's using the port
lsof -i :8000  # or other port number

# Stop conflicting services
docker-compose -f scripts/docker-compose.local.yml down
```

### Connection refused
```bash
# Wait longer for services to start
sleep 10

# Check service health
docker ps
docker-compose -f scripts/docker-compose.local.yml ps
```

## Clean Up

Remove all containers and volumes:
```bash
docker-compose -f scripts/docker-compose.local.yml down -v
```

Remove test configuration files:
```bash
rm /tmp/*-test-config.yaml
rm /tmp/test-file.txt
rm /tmp/downloaded-file.txt
```

## Next Steps

1. Run all tests locally: `./scripts/validate-local.sh`
2. Test individual integrations with test scripts
3. Validate against real AWS services
4. Review test coverage: `go test -cover ./internal/sources/...`
5. Check the production deployment guide (coming soon)

## Support

For issues with:
- **Local emulators:** Check Docker logs and service documentation
- **Go tests:** Review test output and error messages
- **Real AWS services:** Check AWS CloudWatch logs and IAM permissions

See the main project README for more information.
