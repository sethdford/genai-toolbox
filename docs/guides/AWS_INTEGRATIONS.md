# AWS Database Integrations for MCP Toolbox

This document describes all AWS database integrations that have been implemented for the MCP Toolbox for Databases.

## Overview

The following AWS database and data services have been integrated:

1. **Amazon DynamoDB** - NoSQL key-value and document database
2. **Amazon Redshift** - Cloud data warehouse
3. **Amazon DocumentDB** - MongoDB-compatible document database
4. **Amazon Neptune** - Graph database (Gremlin/SPARQL)
5. **Amazon Timestream** - Time series database
6. **Amazon QLDB** - Quantum Ledger Database
7. **Amazon Athena** - Serverless interactive query service
8. **Amazon S3** - Object storage (with query capabilities via Athena)
9. **Tableau** - Business intelligence and analytics platform

## Integration Details

### 1. Amazon DynamoDB

**Source Kind:** `dynamodb`

**Configuration Example:**
```yaml
sources:
  my-dynamodb:
    kind: dynamodb
    region: us-east-1
    # Optional: for DynamoDB Local
    endpoint: http://localhost:8000
    # Optional: explicit credentials
    accessKeyId: YOUR_ACCESS_KEY
    secretAccessKey: YOUR_SECRET_KEY
```

**Features:**
- Full AWS SDK v2 integration
- Support for DynamoDB Local
- Connection pooling and authentication
- IAM role support via AWS default credential chain

**Location:** `/internal/sources/dynamodb/`

---

### 2. Amazon Redshift

**Source Kind:** `redshift`

**Configuration Example:**
```yaml
sources:
  my-redshift:
    kind: redshift
    host: mycluster.abc123.us-west-2.redshift.amazonaws.com
    port: "5439"
    user: admin
    password: mypassword
    database: mydb
    queryParams:
      application_name: genai-toolbox
```

**Features:**
- PostgreSQL-compatible connection
- Connection pooling (max 25 open, 5 idle connections)
- Query parameter support
- User agent tracking

**Location:** `/internal/sources/redshift/`

---

### 3. Amazon DocumentDB

**Source Kind:** `documentdb`

**Configuration Example:**
```yaml
sources:
  my-documentdb:
    kind: documentdb
    uri: mongodb://username:password@mycluster.us-east-1.docdb.amazonaws.com:27017
    tlsCAFile: /path/to/rds-combined-ca-bundle.pem
```

**Features:**
- MongoDB-compatible driver
- TLS/SSL support with CA certificates
- Connection verification via ping
- Application name tracking

**Location:** `/internal/sources/documentdb/`

---

### 4. Amazon Neptune

**Source Kind:** `neptune`

**Configuration Example:**
```yaml
sources:
  my-neptune:
    kind: neptune
    endpoint: wss://your-neptune-endpoint:8182/gremlin
    useIAM: true
```

**Features:**
- Gremlin (TinkerPop) support
- WebSocket connection
- IAM authentication support
- Graph query capabilities

**Location:** `/internal/sources/neptune/`

---

### 5. Amazon Timestream

**Source Kind:** `timestream`

**Configuration Example:**
```yaml
sources:
  my-timestream:
    kind: timestream
    region: us-east-1
    database: myTimeseriesDB
```

**Features:**
- Separate query and write clients
- Time series optimized queries
- AWS SDK v2 integration
- Database listing verification

**Location:** `/internal/sources/timestream/`

---

### 6. Amazon QLDB (Quantum Ledger Database)

**Source Kind:** `qldb`

**Configuration Example:**
```yaml
sources:
  my-qldb:
    kind: qldb
    region: us-east-1
    ledgerName: myLedger
```

**Features:**
- Immutable transaction log
- Cryptographically verifiable
- PartiQL query support
- Session and service clients

**Location:** `/internal/sources/qldb/`

---

### 7. Amazon Athena

**Source Kind:** `athena`

**Configuration Example:**
```yaml
sources:
  my-athena:
    kind: athena
    region: us-east-1
    database: default
    outputLocation: s3://my-bucket/athena-results/
    workGroup: primary
    encryptionOption: SSE_S3
```

**Features:**
- Serverless SQL queries on S3 data
- Multiple encryption options (SSE_S3, SSE_KMS, CSE_KMS)
- Workgroup support
- Query results management

**Location:** `/internal/sources/athena/`

---

### 8. Amazon S3

**Source Kind:** `s3`

**Configuration Example:**
```yaml
sources:
  my-s3:
    kind: s3
    region: us-east-1
    bucket: my-default-bucket
    # Optional: for S3-compatible services (MinIO, etc.)
    endpoint: https://s3.example.com
    forcePathStyle: true
```

**Features:**
- Full S3 API support
- S3-compatible service support (MinIO, etc.)
- Path-style and virtual-hosted-style addressing
- Bucket operations and object management

**Location:** `/internal/sources/s3/`

---

### 9. Tableau

**Source Kind:** `tableau`

**Configuration Example:**
```yaml
sources:
  my-tableau:
    kind: tableau
    serverUrl: https://tableau.example.com
    siteName: MyOrg
    # Option 1: Personal Access Token (recommended)
    personalAccessTokenName: my-token
    personalAccessTokenSecret: token-secret
    # Option 2: Username/Password
    # username: myuser
    # password: mypassword
    apiVersion: "3.19"
```

**Features:**
- REST API integration
- Personal Access Token (PAT) authentication
- Username/password authentication
- Multi-site deployment support
- Configurable API version

**Location:** `/internal/sources/tableau/`

---

## Aurora Database Notes

**Amazon RDS Aurora PostgreSQL** and **Amazon RDS Aurora MySQL** can use the existing source integrations:

- **Aurora PostgreSQL**: Use the existing `postgres` source kind
- **Aurora MySQL**: Use the existing `mysql` source kind

Aurora endpoints are fully compatible with standard PostgreSQL and MySQL drivers, so no separate integration is needed. Simply configure them with your Aurora cluster endpoint.

Example for Aurora PostgreSQL:
```yaml
sources:
  my-aurora-pg:
    kind: postgres
    host: myaurora-cluster.cluster-xxx.us-east-1.rds.amazonaws.com
    port: "5432"
    user: postgres
    password: mypassword
    database: mydb
```

Example for Aurora MySQL:
```yaml
sources:
  my-aurora-mysql:
    kind: mysql
    host: myaurora-cluster.cluster-xxx.us-east-1.rds.amazonaws.com
    port: "3306"
    user: admin
    password: mypassword
    database: mydb
```

---

## ElastiCache and MemoryDB Notes

**Amazon ElastiCache for Redis** and **Amazon MemoryDB for Redis** can use the existing `redis` source integration:

Example:
```yaml
sources:
  my-elasticache:
    kind: redis
    host: my-cluster.xxx.cache.amazonaws.com
    port: "6379"
    password: mypassword
    database: 0
```

---

## Cassandra/Keyspaces Notes

**Amazon Keyspaces (for Apache Cassandra)** can use the existing `cassandra` source integration with appropriate endpoint configuration:

Example:
```yaml
sources:
  my-keyspaces:
    kind: cassandra
    hosts:
      - cassandra.us-east-1.amazonaws.com
    port: 9142
    username: myuser
    password: mypassword
    keyspace: mykeyspace
    # Additional SSL configuration may be needed
```

---

## Authentication

All AWS integrations support multiple authentication methods:

1. **Default AWS Credential Chain** (recommended)
   - IAM roles (for EC2, ECS, Lambda)
   - Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
   - AWS credentials file (`~/.aws/credentials`)
   - AWS CLI configuration

2. **Explicit Credentials** (optional)
   - Specify `accessKeyId` and `secretAccessKey` in configuration
   - Not recommended for production use

3. **IAM Authentication** (where supported)
   - DynamoDB, Timestream, QLDB, Athena, S3
   - Neptune with `useIAM: true`

---

## Required Dependencies

To use these integrations, ensure your `go.mod` includes:

```go
require (
    github.com/aws/aws-sdk-go-v2 v1.x.x
    github.com/aws/aws-sdk-go-v2/config v1.x.x
    github.com/aws/aws-sdk-go-v2/service/dynamodb v1.x.x
    github.com/aws/aws-sdk-go-v2/service/athena v1.x.x
    github.com/aws/aws-sdk-go-v2/service/s3 v1.x.x
    github.com/aws/aws-sdk-go-v2/service/timestreamquery v1.x.x
    github.com/aws/aws-sdk-go-v2/service/timestreamwrite v1.x.x
    github.com/aws/aws-sdk-go-v2/service/qldb v1.x.x
    github.com/aws/aws-sdk-go-v2/service/qldbsession v1.x.x
    github.com/lib/pq v1.x.x  // For Redshift (PostgreSQL driver)
    go.mongodb.org/mongo-driver v1.x.x  // For DocumentDB
    github.com/apache/tinkerpop/gremlin-go/v3 v3.x.x  // For Neptune
)
```

---

## Testing

Test files are included for each integration:
- `dynamodb_test.go`
- `redshift_test.go`
- More tests to be added

Integration tests require:
- Valid AWS credentials
- Access to the respective AWS services
- Appropriate IAM permissions

---

## Next Steps

1. **Add Tool Implementations**: Create tool definitions for common operations on each database
2. **Enhanced Testing**: Add comprehensive integration tests
3. **Documentation**: Add usage examples and best practices
4. **MCP Tool Definitions**: Create `tools.yaml` examples for each source type

---

## Contributing

When adding new AWS integrations:

1. Follow the existing pattern in `/internal/sources/`
2. Implement the `SourceConfig` and `Source` interfaces
3. Register the source in the `init()` function
4. Add comprehensive tests
5. Update this documentation

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/googleapis/genai-toolbox/issues
- Discord Community: https://discord.gg/Dmm69peqjh
