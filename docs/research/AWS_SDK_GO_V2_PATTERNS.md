# AWS SDK Go v2 - Complete Reference Guide

**Last Updated:** November 2024
**SDK Version:** AWS SDK for Go v2
**Minimum Go Version:** 1.23+

---

## Table of Contents

1. [Authentication & Credentials](#authentication--credentials)
2. [Service Client Configuration](#service-client-configuration)
3. [Connection Management](#connection-management)
4. [Service-Specific Patterns](#service-specific-patterns)
5. [Common Gotchas & Mistakes](#common-gotchas--mistakes)
6. [Best Practices](#best-practices)

---

## Authentication & Credentials

### Credential Chain Order and Precedence

When using `config.LoadDefaultConfig()`, the SDK searches for credentials in the following order:

1. **Programmatically provided options** (highest priority)
   - Options passed to `config.LoadDefaultConfig()`
   - `config.WithCredentialsProvider()`

2. **Environment Variables**
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
   - `AWS_SESSION_TOKEN` (optional, for temporary credentials)

3. **Shared Configuration Files**
   - `~/.aws/credentials` (credentials file - takes precedence)
   - `~/.aws/config` (config file)
   - If a profile exists in both files, credentials file properties take precedence

4. **IAM Role for ECS Tasks**
   - If running on Amazon ECS

5. **IAM Role for EC2**
   - If running on an Amazon EC2 instance

**Official Documentation:**
- https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-auth.html
- https://docs.aws.amazon.com/sdkref/latest/guide/standardized-credentials.html

### Using LoadDefaultConfig (Recommended)

```go
package main

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
)

func main() {
    // Load default configuration
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        panic("configuration error, " + err.Error())
    }

    // Use cfg to create service clients
}
```

### LoadDefaultConfig with Options

```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-west-2"),
    config.WithSharedConfigProfile("customProfile"),
    config.WithRetryMode(aws.RetryModeStandard),
    config.WithRetryMaxAttempts(5),
)
```

### Static Credentials (Explicit Credentials)

Use `credentials.NewStaticCredentialsProvider` when you need to inject explicit credentials:

```go
package main

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
    // Create static credentials provider
    staticProvider := credentials.NewStaticCredentialsProvider(
        "AKIAIOSFODNN7EXAMPLE",     // Access Key ID
        "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", // Secret Access Key
        "",                          // Session Token (empty if not using temporary credentials)
    )

    // Load config with static credentials
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(staticProvider),
    )
    if err != nil {
        panic(err)
    }
}
```

**Function Signature:**
```go
func NewStaticCredentialsProvider(key, secret, session string) StaticCredentialsProvider
```

**When to Use:**
- Testing/development environments
- DynamoDB Local or LocalStack
- When credentials come from custom sources (secrets manager, vault, etc.)

**Security Note:** Never embed credentials in source code for production. Use environment variables or AWS IAM roles.

### Session Tokens (Temporary Credentials)

Session tokens are used with temporary credentials from AWS STS:

```go
staticProvider := credentials.NewStaticCredentialsProvider(
    accessKey,
    secretKey,
    sessionToken, // Include the session token
)

cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithCredentialsProvider(staticProvider),
)
```

### IAM Role Assumption (AssumeRole)

Use `stscreds.NewAssumeRoleProvider` to assume an IAM role:

```go
package main

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
    // Load default config for initial credentials
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        panic(err)
    }

    // Create STS client
    stsClient := sts.NewFromConfig(cfg)

    // Create AssumeRole credentials provider
    roleProvider := stscreds.NewAssumeRoleProvider(
        stsClient,
        "arn:aws:iam::123456789012:role/MyRole",
    )

    // Create new config with assumed role credentials
    cfg.Credentials = aws.NewCredentialsCache(roleProvider)

    // Use the config with assumed role credentials
    s3Client := s3.NewFromConfig(cfg)
}
```

### AssumeRole with MFA

```go
roleProvider := stscreds.NewAssumeRoleProvider(
    stsClient,
    "arn:aws:iam::123456789012:role/MyRole",
    func(o *stscreds.AssumeRoleOptions) {
        o.SerialNumber = aws.String("arn:aws:iam::123456789012:mfa/user")
        o.TokenProvider = stscreds.StdinTokenProvider
        o.RoleSessionName = "my-session"
        o.Duration = 3600 * time.Second // 1 hour
    },
)
```

**AssumeRoleOptions:**
- `RoleARN` (required): The IAM role to assume
- `RoleSessionName`: Unique session identifier
- `Duration`: Credential validity period (default 15 minutes)
- `ExternalID`: Optional external identifier
- `Policy`: Optional IAM policy constraints
- `SerialNumber`: MFA device identification
- `TokenProvider`: Method for providing MFA tokens

**Official Documentation:**
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/stscreds
- https://docs.aws.amazon.com/code-library/latest/ug/sts_example_sts_AssumeRole_section.html

---

## Service Client Configuration

### Creating Service Clients

Service clients are created using `NewFromConfig`:

```go
package main

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        panic(err)
    }

    // Create service clients
    dynamoClient := dynamodb.NewFromConfig(cfg)
    s3Client := s3.NewFromConfig(cfg)
}
```

### Region Configuration

**IMPORTANT:** AWS SDK for Go v2 does NOT have a default region. You MUST specify a region.

```go
// Option 1: Environment variable
// AWS_REGION=us-east-1

// Option 2: During config load
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-west-2"),
)

// Option 3: Default region (fallback if not set elsewhere)
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithDefaultRegion("us-east-1"),
)
```

### Endpoint Configuration

#### Custom Endpoint (Simple Method)

```go
// For DynamoDB Local, LocalStack, or custom endpoints
customEndpoint := "http://localhost:8000"

client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.BaseEndpoint = &customEndpoint
})
```

#### Custom Endpoint Resolver (Advanced)

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
)

cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithEndpointResolverWithOptions(
        aws.EndpointResolverWithOptionsFunc(
            func(service, region string, options ...interface{}) (aws.Endpoint, error) {
                if service == dynamodb.ServiceID {
                    return aws.Endpoint{
                        URL: "http://localhost:8000",
                    }, nil
                }
                // Fallback to default resolver
                return aws.Endpoint{}, &aws.EndpointNotFoundError{}
            },
        ),
    ),
)
```

**Official Documentation:**
- https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html

---

## Connection Management

### Do AWS SDK v2 Clients Need Close()?

**NO** - Service clients do NOT need to be closed or cleaned up.

```go
// This is CORRECT - no cleanup needed
client := s3.NewFromConfig(cfg)
// Use client...
// NO need to call client.Close()
```

**Important:** Service clients are designed to be:
- **Long-lived** - Create once, reuse throughout application lifetime
- **Thread-safe** - Safe to use concurrently across goroutines
- **Stateful** - Contains credentials cache, retry token bucket, and connection pool

### What DOES Need to Be Closed?

**CRITICAL:** You MUST close `io.ReadCloser` response bodies:

```go
// Amazon S3 GetObject example
resp, err := client.GetObject(ctx, &s3.GetObjectInput{
    Bucket: aws.String("mybucket"),
    Key:    aws.String("mykey"),
})
if err != nil {
    return err
}

// MUST close the body, even if not reading it
defer resp.Body.Close()

// Read the body
data, err := io.ReadAll(resp.Body)
```

**Why?** Failure to close response bodies can:
- Leak connections
- Cause connection pool exhaustion
- Result in connection reset errors on subsequent requests

### Connection Pooling

AWS SDK v2 uses Go's standard `http.Transport` with automatic connection pooling.

**Default Connection Pool Settings:**
```go
// Default values (you don't need to set these)
DefaultHTTPTransportMaxIdleConns         = 100
DefaultHTTPTransportMaxIdleConnsPerHost  = 10
DefaultHTTPTransportIdleConnTimeout      = 90 * time.Second
DefaultHTTPTransportTLSHandshakeTimeout  = 10 * time.Second
DefaultHTTPTransportExpectContinueTimeout = 1 * time.Second
```

**Connection Pooling Behavior:**
- **Automatic** - HTTP KeepAlive is enabled by default
- **Per-client** - Each service client has its own connection pool
- **Not shared** - Copying a client creates a new connection pool

### Custom HTTP Client Configuration

For high-throughput applications, you may need custom HTTP client settings:

```go
import (
    "net"
    "net/http"
    "time"
    awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
)

httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
    tr.MaxIdleConns = 200
    tr.MaxIdleConnsPerHost = 50
    tr.IdleConnTimeout = 120 * time.Second
    tr.DialContext = (&net.Dialer{
        Timeout:   30 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext
})

cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithHTTPClient(httpClient),
)
```

**Official Documentation:**
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/transport/http
- https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-http.html

### Context Handling and Timeouts

All SDK operations accept a `context.Context` for timeout and cancellation:

```go
import (
    "context"
    "time"
)

// Set timeout for individual operation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.GetObject(ctx, &s3.GetObjectInput{
    Bucket: aws.String("mybucket"),
    Key:    aws.String("mykey"),
})
if err != nil {
    // Check if context deadline exceeded
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Operation timed out")
    }
    return err
}
defer resp.Body.Close()
```

### Retry Configuration

**Default Retry Behavior:**
- Retryer: `retry.Standard`
- Max attempts: 3
- Max backoff delay: 20 seconds
- Client-side rate limiting: Enabled (token bucket)

**Limit Max Attempts:**
```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRetryer(func() aws.Retryer {
        return retry.AddWithMaxAttempts(retry.NewStandard(), 5)
    }),
)
```

**Limit Backoff Delay:**
```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRetryer(func() aws.Retryer {
        return retry.AddWithMaxBackoffDelay(retry.NewStandard(), 5*time.Second)
    }),
)
```

**Disable Retries:**
```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRetryer(func() aws.Retryer {
        return aws.NopRetryer{}
    }),
)
```

**IMPORTANT:** Setting max attempts to zero allows infinite retries, which can cause runaway workloads and inflated billing.

**Retry and Context Interaction:**
The standard retryer will NOT retry if the context is cancelled. Your application must handle this explicitly.

**Official Documentation:**
- https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/retry

---

## Service-Specific Patterns

### DynamoDB

#### DynamoDB Client Best Practices

```go
import (
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Create client
cfg, err := config.LoadDefaultConfig(context.TODO())
dynamoClient := dynamodb.NewFromConfig(cfg)

// Use client for operations
result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
    TableName: aws.String("MyTable"),
    Key: map[string]types.AttributeValue{
        "id": &types.AttributeValueMemberS{Value: "123"},
    },
})
```

#### DynamoDB Local Configuration

```go
// Method 1: Using BaseEndpoint (Recommended)
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
)

localEndpoint := "http://localhost:8000"
dynamoClient := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.BaseEndpoint = &localEndpoint
})

// Method 2: Using EndpointResolverWithOptions
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
    config.WithEndpointResolverWithOptions(
        aws.EndpointResolverWithOptionsFunc(
            func(service, region string, options ...interface{}) (aws.Endpoint, error) {
                return aws.Endpoint{
                    URL: "http://localhost:8000",
                }, nil
            },
        ),
    ),
    config.WithCredentialsProvider(
        credentials.StaticCredentialsProvider{
            Value: aws.Credentials{
                AccessKeyID:     "dummy",
                SecretAccessKey: "dummy",
                SessionToken:    "dummy",
                Source:          "Hard-coded credentials for local DynamoDB",
            },
        },
    ),
)

dynamoClient := dynamodb.NewFromConfig(cfg)
```

**Key Points:**
- DynamoDB Local runs on `http://localhost:8000` by default
- Dummy credentials are sufficient (values don't matter for local)
- Region must still be specified even for local development

**Official Documentation:**
- https://docs.aws.amazon.com/code-library/latest/ug/go_2_dynamodb_code_examples.html
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/dynamodb

### S3

#### S3 Client Configuration

```go
import (
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

cfg, err := config.LoadDefaultConfig(context.TODO())
s3Client := s3.NewFromConfig(cfg)
```

#### Path-Style vs Virtual-Hosted-Style

**Virtual-Hosted-Style (Default):**
```
https://mybucket.s3.amazonaws.com/mykey
```

**Path-Style:**
```
https://s3.amazonaws.com/mybucket/mykey
```

**Configure Path-Style:**
```go
s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.UsePathStyle = true
})
```

**When to Use Path-Style:**
- LocalStack or S3-compatible services (MinIO, Ceph)
- Custom S3 endpoints
- Legacy applications

**IMPORTANT:** Directory buckets MUST use virtual-hosted-style. Path-style is not supported for directory buckets.

#### S3 with Custom Endpoint

```go
// For LocalStack, MinIO, or S3-compatible storage
customEndpoint := "http://localhost:4566"

s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = &customEndpoint
    o.UsePathStyle = true  // Often required for S3-compatible services
})
```

#### S3 Best Practices

**CRITICAL:** Always close S3 GetObject response bodies:

```go
resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
    Bucket: aws.String("mybucket"),
    Key:    aws.String("mykey"),
})
if err != nil {
    return err
}
defer resp.Body.Close()  // MUST close

// Read body
data, err := io.ReadAll(resp.Body)
```

**Consume Response Bodies Quickly:**
Delayed consumption can cause connection resets. Read the body promptly or stream it to destination.

**Official Documentation:**
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
- https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html

### Athena

#### Athena Query Pattern

Athena queries follow a three-step pattern:

1. **Start query execution** - Returns query execution ID
2. **Poll query status** - Wait for completion
3. **Get query results** - Retrieve result set

```go
import (
    "github.com/aws/aws-sdk-go-v2/service/athena"
    "github.com/aws/aws-sdk-go-v2/service/athena/types"
)

// 1. Start query execution
startResp, err := athenaClient.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
    QueryString: aws.String("SELECT * FROM my_table LIMIT 10"),
    ResultConfiguration: &types.ResultConfiguration{
        OutputLocation: aws.String("s3://my-bucket/athena-results/"),
    },
    QueryExecutionContext: &types.QueryExecutionContext{
        Database: aws.String("my_database"),
    },
})
if err != nil {
    return err
}

queryExecutionID := startResp.QueryExecutionId

// 2. Poll for completion
for {
    statusResp, err := athenaClient.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
        QueryExecutionId: queryExecutionID,
    })
    if err != nil {
        return err
    }

    state := statusResp.QueryExecution.Status.State

    switch state {
    case types.QueryExecutionStateSucceeded:
        goto GetResults
    case types.QueryExecutionStateFailed, types.QueryExecutionStateCancelled:
        return fmt.Errorf("query failed: %s", *statusResp.QueryExecution.Status.StateChangeReason)
    }

    time.Sleep(1 * time.Second)
}

GetResults:
// 3. Get query results
resultsResp, err := athenaClient.GetQueryResults(ctx, &athena.GetQueryResultsInput{
    QueryExecutionId: queryExecutionID,
})
if err != nil {
    return err
}

// Process results
for _, row := range resultsResp.ResultSet.Rows {
    // Process row data
}
```

**Important Notes:**
- `GetQueryResults` does NOT execute queries - use `StartQueryExecution` first
- Results are paginated - use `NextToken` for large result sets
- Results are stored in S3 at the `OutputLocation`
- IAM principal needs permissions for both `athena:GetQueryResults` and `s3:GetObject`

**Official Documentation:**
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/athena
- https://docs.aws.amazon.com/athena/latest/APIReference/

### Timestream

#### Timestream Dual-Client Usage

Timestream requires TWO separate clients:
- **Write Client** - For inserting and managing time series data
- **Query Client** - For querying time series data

```go
import (
    "github.com/aws/aws-sdk-go-v2/service/timestreamquery"
    "github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
)

cfg, err := config.LoadDefaultConfig(context.TODO())

// Create Write client for data ingestion
writeClient := timestreamwrite.NewFromConfig(cfg)

// Create Query client for data retrieval
queryClient := timestreamquery.NewFromConfig(cfg)

// Write data
_, err = writeClient.WriteRecords(ctx, &timestreamwrite.WriteRecordsInput{
    DatabaseName: aws.String("myDatabase"),
    TableName:    aws.String("myTable"),
    Records:      records,
})

// Query data
queryResp, err := queryClient.Query(ctx, &timestreamquery.QueryInput{
    QueryString: aws.String("SELECT * FROM myDatabase.myTable"),
})
```

**Key Points:**
- Write SDK: CRUD operations, data insertion
- Query SDK: Query execution
- Both clients can use the same `aws.Config`

**Official Documentation:**
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/timestreamquery
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/timestreamwrite
- https://docs.aws.amazon.com/timestream/latest/developerguide/getting-started-sdks.html

### QLDB

#### QLDB Session Management

**RECOMMENDATION:** Use the QLDB driver instead of the low-level session API.

The QLDB driver provides high-level abstraction and manages `SendCommand` API calls automatically.

**Low-Level Session API (Not Recommended):**
```go
import (
    "github.com/aws/aws-sdk-go-v2/service/qldbsession"
)

sessionClient := qldbsession.NewFromConfig(cfg)

// SendCommand operations
resp, err := sessionClient.SendCommand(ctx, &qldbsession.SendCommandInput{
    StartSession: &types.StartSessionRequest{
        LedgerName: aws.String("myLedger"),
    },
})
```

**Official Documentation:**
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/qldbsession
- https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/qldb

---

## Common Gotchas & Mistakes

### 1. Not Specifying a Region

**MISTAKE:**
```go
// This will fail - no region specified
cfg, err := config.LoadDefaultConfig(context.TODO())
```

**ERROR:** Missing region in config

**FIX:**
```go
// Set environment variable
// AWS_REGION=us-east-1

// OR specify in code
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
)
```

### 2. Not Closing Response Bodies

**MISTAKE:**
```go
resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{...})
// Missing: defer resp.Body.Close()
data, _ := io.ReadAll(resp.Body)
```

**CONSEQUENCE:** Connection leaks, connection pool exhaustion, connection reset errors

**FIX:**
```go
resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{...})
if err != nil {
    return err
}
defer resp.Body.Close()  // ALWAYS close
data, _ := io.ReadAll(resp.Body)
```

### 3. Incorrect Error Handling

**MISTAKE:**
```go
_, err := client.GetObject(ctx, input)
if err.Error() == "NoSuchKey" {  // String comparison - BAD
    // Handle
}
```

**FIX:** Use `errors.As` to check error types:
```go
import (
    "errors"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

_, err := client.GetObject(ctx, input)
if err != nil {
    var nsk *types.NoSuchKey
    if errors.As(err, &nsk) {
        // Handle NoSuchKey error
    }
}
```

### 4. Embedding Credentials in Code

**MISTAKE:**
```go
// NEVER do this in production
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithCredentialsProvider(
        credentials.NewStaticCredentialsProvider(
            "AKIAIOSFODNN7EXAMPLE",
            "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
            "",
        ),
    ),
)
```

**FIX:**
- Use environment variables
- Use IAM roles (EC2, ECS, Lambda)
- Use AWS IAM Identity Center
- Load from secure secret managers

### 5. Creating Clients in Hot Paths

**MISTAKE:**
```go
// Creating new client on every request
func handleRequest(w http.ResponseWriter, r *http.Request) {
    cfg, _ := config.LoadDefaultConfig(context.TODO())  // SLOW
    client := s3.NewFromConfig(cfg)  // SLOW
    // Use client...
}
```

**FIX:** Create clients once, reuse them:
```go
// Create once at startup
var s3Client *s3.Client

func init() {
    cfg, _ := config.LoadDefaultConfig(context.TODO())
    s3Client = s3.NewFromConfig(cfg)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Reuse client - safe for concurrent use
}
```

### 6. Infinite Retries

**MISTAKE:**
```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRetryer(func() aws.Retryer {
        return retry.AddWithMaxAttempts(retry.NewStandard(), 0)  // Infinite retries
    }),
)
```

**CONSEQUENCE:** Runaway workloads, inflated billing, hung requests

**FIX:**
```go
// Set reasonable max attempts
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRetryer(func() aws.Retryer {
        return retry.AddWithMaxAttempts(retry.NewStandard(), 5)
    }),
)
```

### 7. DynamoDB Local Without Region

**MISTAKE:**
```go
// Missing region for DynamoDB Local
cfg, err := config.LoadDefaultConfig(context.TODO())
localEndpoint := "http://localhost:8000"
client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.BaseEndpoint = &localEndpoint
})
```

**FIX:**
```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),  // Region still required
)
localEndpoint := "http://localhost:8000"
client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.BaseEndpoint = &localEndpoint
})
```

### 8. S3 Path-Style with Directory Buckets

**MISTAKE:**
```go
// Directory buckets don't support path-style
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.UsePathStyle = true  // Will fail for directory buckets
})
```

**FIX:** Use virtual-hosted-style (default) for directory buckets.

### 9. Not Handling Context Cancellation

**MISTAKE:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_, err := client.GetObject(ctx, input)
// Not checking if timeout caused the error
```

**FIX:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_, err := client.GetObject(ctx, input)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Operation timed out")
    }
    return err
}
```

### 10. Modifying Endpoint URLs with Proxies

**GOTCHA:** Proxies may modify request headers (especially `X-*` headers), breaking SigV4 signature validation.

**FIX:** Configure proxy to not modify AWS-specific headers, or capture outgoing requests for debugging.

---

## Best Practices

### 1. Client Lifecycle

**DO:**
- Create service clients ONCE at application startup
- Reuse clients throughout application lifetime
- Clients are thread-safe and designed for concurrent use

**DON'T:**
- Create new clients on every request
- Copy clients unnecessarily (creates new connection pools)
- Try to close or cleanup clients

### 2. Credentials Management

**DO:**
- Use IAM roles for EC2, ECS, Lambda (best security)
- Use environment variables for local development
- Store sensitive values in `~/.aws/credentials`, not `~/.aws/config`

**DON'T:**
- Embed credentials in source code
- Commit credentials to version control
- Use root account credentials

### 3. Error Handling

**DO:**
- Use `errors.As()` to check specific error types
- Check `errors.Unwrap()` for underlying errors
- Handle context cancellation explicitly

**DON'T:**
- Compare error strings
- Ignore error details
- Assume all errors are retryable

### 4. Performance

**DO:**
- Consume response bodies quickly
- ALWAYS close `io.ReadCloser` instances
- Use appropriate timeouts via context
- Configure retry limits
- For high throughput, tune HTTP client connection pool

**DON'T:**
- Leave response bodies unclosed
- Use infinite retries
- Create clients in hot paths
- Delay reading response bodies

### 5. Configuration

**DO:**
- Specify region explicitly (no default in v2)
- Use functional options for service-specific config
- Use `config.WithDefaultRegion()` as fallback
- Validate configuration at startup

**DON'T:**
- Assume a default region exists
- Mix v1 and v2 SDK patterns
- Ignore configuration errors

### 6. Testing

**DO:**
- Use `BaseEndpoint` for local services (DynamoDB Local, LocalStack)
- Use static credentials for testing
- Mock service clients for unit tests
- Test error handling paths

**DON'T:**
- Use production credentials in tests
- Skip error handling tests
- Assume local services work exactly like AWS

### 7. Observability

**DO:**
- Use SDK timing telemetry for performance diagnosis
- Log credential source for debugging
- Monitor retry rates
- Track context timeouts

**DON'T:**
- Log sensitive credential data
- Ignore retry metrics
- Overlook connection pool exhaustion

---

## Migration from SDK v1 to v2

### Key Differences

1. **No Default Region:** v2 requires explicit region configuration
2. **Modular Design:** Each service is an independent Go module
3. **Unified Config:** Single `aws.Config` type instead of Session + Config
4. **Context-First:** All operations require `context.Context`
5. **Error Handling:** Errors implement `Unwrap()` for better error chains

### Basic Migration Pattern

**v1:**
```go
sess := session.Must(session.NewSession())
svc := s3.New(sess)
```

**v2:**
```go
cfg, err := config.LoadDefaultConfig(context.TODO())
if err != nil {
    panic(err)
}
svc := s3.NewFromConfig(cfg)
```

**Official Migration Guide:**
- https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/migrate-gosdk.html

---

## Official Resources

### Documentation
- **Developer Guide:** https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/
- **API Reference:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2
- **GitHub Repository:** https://github.com/aws/aws-sdk-go-v2
- **Code Examples:** https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/go_code_examples.html

### Key Packages
- **Config:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config
- **Credentials:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials
- **STS Credentials:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/stscreds
- **Retry:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/retry
- **HTTP Transport:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/transport/http

### Service-Specific Packages
- **DynamoDB:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/dynamodb
- **S3:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
- **Athena:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/athena
- **Timestream Query:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/timestreamquery
- **Timestream Write:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/timestreamwrite
- **QLDB:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/qldb
- **QLDB Session:** https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/qldbsession

---

## Quick Reference

### Minimal Working Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    // Load config with region
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
    )
    if err != nil {
        panic(err)
    }

    // Create S3 client
    client := s3.NewFromConfig(cfg)

    // List buckets
    result, err := client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
    if err != nil {
        panic(err)
    }

    for _, bucket := range result.Buckets {
        fmt.Printf("Bucket: %s\n", aws.ToString(bucket.Name))
    }
}
```

### Common Configuration Template

```go
package main

import (
    "context"
    "time"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/aws/retry"
)

func createAWSConfig() (aws.Config, error) {
    return config.LoadDefaultConfig(context.TODO(),
        // Region (required)
        config.WithRegion("us-east-1"),

        // Retry configuration
        config.WithRetryer(func() aws.Retryer {
            return retry.AddWithMaxAttempts(
                retry.AddWithMaxBackoffDelay(
                    retry.NewStandard(),
                    10*time.Second,
                ),
                5,
            )
        }),

        // Optional: Custom profile
        // config.WithSharedConfigProfile("myprofile"),

        // Optional: Custom credentials
        // config.WithCredentialsProvider(myCredsProvider),
    )
}
```

---

## Version Information

- **SDK Version:** AWS SDK for Go v2
- **Minimum Go Version:** 1.23+
- **v1 End of Support:** July 31, 2025
- **Status:** General Availability (GA)

---

## Contributing

This document is based on official AWS documentation and community best practices as of November 2024. For the most up-to-date information, always refer to the official AWS SDK for Go v2 documentation.

**Report Issues:** https://github.com/aws/aws-sdk-go-v2/issues

---

## License

This documentation is provided for reference purposes. AWS SDK for Go v2 is licensed under Apache License 2.0.
