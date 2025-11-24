# AWS SDK Go v2 Production Patterns Analysis

Based on analysis of 5+ high-quality repositories and official AWS documentation, this document outlines production-quality patterns for AWS SDK Go v2 implementations.

## Sources Analyzed

1. **Official AWS SDK Go v2 Repository** - https://github.com/aws/aws-sdk-go-v2
2. **AWS Documentation Examples (gov2)** - https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2
3. **HashiCorp aws-sdk-go-base** - https://github.com/hashicorp/aws-sdk-go-base
4. **Terraform AWS Provider** - Production use of SDK in large-scale infrastructure
5. **MinIO Integration Examples** - Custom endpoint configuration patterns

---

## 1. Client Initialization Patterns

### 1.1 Basic Client Initialization

**Pattern: Wrapper Struct**
```go
type BucketBasics struct {
    S3Client *s3.Client
}

type TableBasics struct {
    DynamoDbClient *dynamodb.Client
    TableName      string
}
```

**Benefits:**
- Encapsulation of service clients
- Easy method attachment
- Cleaner dependency injection
- Simplified testing

### 1.2 Configuration Loading

**Pattern: Default Configuration**
```go
import (
    "context"
    "log"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func initializeClients(ctx context.Context) (*dynamodb.Client, *s3.Client, error) {
    // Load default configuration (handles credentials automatically)
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion("us-west-2"),
    )
    if err != nil {
        return nil, nil, fmt.Errorf("unable to load SDK config: %w", err)
    }

    // Create service clients
    dynamoClient := dynamodb.NewFromConfig(cfg)
    s3Client := s3.NewFromConfig(cfg)

    return dynamoClient, s3Client, nil
}
```

**Credential Provider Chain (automatic):**
1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
2. Shared configuration files (~/.aws/config, ~/.aws/credentials)
3. ECS task role credentials
4. EC2 instance role credentials

### 1.3 Custom Endpoint Configuration (MinIO, LocalStack)

**Pattern 1: BaseEndpoint (Simplest)**
```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func newMinIOClient(ctx context.Context, endpoint string) (*s3.Client, error) {
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""),
        ),
    )
    if err != nil {
        return nil, err
    }

    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(endpoint)
        o.UsePathStyle = true  // Important for MinIO
    })

    return client, nil
}
```

**Pattern 2: EndpointResolverWithOptions (Path-Style)**
```go
const defaultRegion = "us-east-1"

type customResolver struct {
    endpoint string
}

func (r *customResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
    return aws.Endpoint{
        PartitionID:       "aws",
        SigningRegion:     defaultRegion,
        URL:               r.endpoint,
        HostnameImmutable: true,  // Prevents virtual-hosted style
    }, nil
}

func newS3ClientWithCustomEndpoint(ctx context.Context, endpoint string) (*s3.Client, error) {
    resolver := &customResolver{endpoint: endpoint}

    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(defaultRegion),
        config.WithEndpointResolverWithOptions(resolver),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider("access_key", "secret_key", ""),
        ),
    )
    if err != nil {
        return nil, err
    }

    return s3.NewFromConfig(cfg), nil
}
```

**Key Points:**
- `HostnameImmutable: true` is crucial for path-style addressing
- MinIO requires path-style URLs: `http://localhost:9000/bucket/key`
- Without it, SDK uses virtual-hosted style: `http://bucket.localhost:9000/key`

### 1.4 HTTP Client Configuration

**Pattern: Production HTTP Client**
```go
import (
    "net/http"
    "time"
    awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
    "github.com/aws/aws-sdk-go-v2/config"
)

func newProductionConfig(ctx context.Context) (aws.Config, error) {
    // Custom HTTP client with connection pooling
    httpClient := awshttp.NewBuildableClient().
        WithTimeout(30 * time.Second).
        WithTransportOptions(func(tr *http.Transport) {
            // Connection pooling
            tr.MaxIdleConns = 100
            tr.MaxIdleConnsPerHost = 10
            tr.IdleConnTimeout = 90 * time.Second

            // Timeouts
            tr.TLSHandshakeTimeout = 10 * time.Second
            tr.ResponseHeaderTimeout = 10 * time.Second
            tr.ExpectContinueTimeout = 1 * time.Second

            // Keep-alive
            tr.DialContext = (&net.Dialer{
                Timeout:   30 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext
        })

    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithHTTPClient(httpClient),
    )

    return cfg, err
}
```

**Default Values:**
- `MaxIdleConns`: 100
- `MaxIdleConnsPerHost`: 10
- `IdleConnTimeout`: 90s
- `TLSHandshakeTimeout`: 10s

---

## 2. Error Handling Patterns

### 2.1 OperationError Wrapping

All SDK errors are wrapped with `smithy.OperationError`:

```go
import (
    "errors"
    "log"
    "github.com/aws/smithy-go"
)

func handleSDKError(err error) {
    if err == nil {
        return
    }

    var oe *smithy.OperationError
    if errors.As(err, &oe) {
        log.Printf("Service: %s, Operation: %s, Error: %v",
            oe.Service(), oe.Operation(), oe.Unwrap())
    }
}
```

### 2.2 Typed Error Handling

**Pattern: Specific Error Types**
```go
import (
    "errors"
    "log"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/smithy-go"
)

func handleDynamoDBError(err error) {
    if err == nil {
        return
    }

    // Check for specific resource not found
    var notFoundEx *types.ResourceNotFoundException
    if errors.As(err, &notFoundEx) {
        log.Printf("Resource not found: %v", notFoundEx)
        return
    }

    // Check for API errors with error codes
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "ConditionalCheckFailedException":
            log.Printf("Condition check failed")
        case "ProvisionedThroughputExceededException":
            log.Printf("Throttled - need backoff")
        case "ValidationException":
            log.Printf("Invalid input: %v", apiErr.ErrorMessage())
        default:
            log.Printf("API error [%s]: %v", apiErr.ErrorCode(), apiErr.ErrorMessage())
        }
        return
    }

    // Generic error
    log.Printf("Unexpected error: %v", err)
}
```

### 2.3 S3 Error Handling

```go
import (
    "errors"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
    "github.com/aws/smithy-go"
)

func handleS3Error(err error) error {
    if err == nil {
        return nil
    }

    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "NoSuchBucket":
            return fmt.Errorf("bucket does not exist: %w", err)
        case "NoSuchKey":
            return fmt.Errorf("object not found: %w", err)
        case "AccessDenied":
            return fmt.Errorf("permission denied: %w", err)
        case "InvalidAccessKeyId":
            return fmt.Errorf("invalid credentials: %w", err)
        }
    }

    var notFound *types.NotFound
    if errors.As(err, &notFound) {
        return fmt.Errorf("resource not found: %w", err)
    }

    return err
}
```

### 2.4 Error Logging Pattern (from AWS Examples)

```go
func (basics *TableBasics) ListTables(ctx context.Context) ([]string, error) {
    var tableNames []string

    paginator := dynamodb.NewListTablesPaginator(basics.DynamoDbClient, &dynamodb.ListTablesInput{})

    for paginator.HasMorePages() {
        output, err := paginator.NextPage(ctx)
        if err != nil {
            log.Printf("Couldn't list tables. Here's why: %v\n", err)
            return nil, err
        }
        tableNames = append(tableNames, output.TableNames...)
    }

    return tableNames, nil
}
```

**Key Pattern:**
- Log context with "Here's why: %v"
- Return error for caller to handle
- Provide operation-specific context in log message

---

## 3. Resource Cleanup Patterns

### 3.1 Response Body Cleanup (S3)

**Critical Pattern:**
```go
func (basics *BucketBasics) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
    result, err := basics.S3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, fmt.Errorf("couldn't get object: %w", err)
    }
    defer result.Body.Close()  // CRITICAL: Always close response bodies

    body, err := io.ReadAll(result.Body)
    if err != nil {
        return nil, fmt.Errorf("couldn't read object body: %w", err)
    }

    return body, nil
}
```

### 3.2 File Handling

**Pattern: Multiple Defers**
```go
func (basics *BucketBasics) UploadFile(ctx context.Context, bucket, key, fileName string) error {
    file, err := os.Open(fileName)
    if err != nil {
        return fmt.Errorf("couldn't open file %s: %w", fileName, err)
    }
    defer file.Close()  // First defer

    _, err = basics.S3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
        Body:   file,
    })
    if err != nil {
        return fmt.Errorf("couldn't upload file: %w", err)
    }

    return nil
}

func (basics *BucketBasics) DownloadFile(ctx context.Context, bucket, key, fileName string) error {
    result, err := basics.S3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("couldn't get object: %w", err)
    }
    defer result.Body.Close()  // First defer

    file, err := os.Create(fileName)
    if err != nil {
        return fmt.Errorf("couldn't create file %s: %w", fileName, err)
    }
    defer file.Close()  // Second defer

    _, err = io.Copy(file, result.Body)
    if err != nil {
        return fmt.Errorf("couldn't write to file: %w", err)
    }

    return nil
}
```

### 3.3 SDK Clients - No Close Method

**Important:** AWS SDK v2 clients do NOT have a Close() method. They are designed to be:
- Created once and reused
- Thread-safe for concurrent use
- Garbage collected automatically

**Anti-pattern (DO NOT DO):**
```go
// WRONG - no Close() method exists
defer s3Client.Close()  // This doesn't exist!
```

**Correct pattern:**
```go
// Create once, use throughout application lifecycle
var (
    s3Client      *s3.Client
    dynamoClient  *dynamodb.Client
)

func init() {
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    s3Client = s3.NewFromConfig(cfg)
    dynamoClient = dynamodb.NewFromConfig(cfg)
}
```

### 3.4 Context Cleanup

**Pattern: Context with Timeout**
```go
func operationWithTimeout(timeout time.Duration) error {
    ctx := context.Background()
    var cancelFn func()

    if timeout > 0 {
        ctx, cancelFn = context.WithTimeout(ctx, timeout)
        defer cancelFn()  // IMPORTANT: Prevent context leak
    }

    _, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("my-key"),
        Body:   bytes.NewReader(data),
    })

    return err
}
```

---

## 4. Context and Timeout Patterns

### 4.1 Basic Context Usage

```go
import (
    "context"
    "time"
)

// Simple operation with background context
func simpleOperation() error {
    ctx := context.Background()

    _, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String("my-table"),
        Key: map[string]types.AttributeValue{
            "id": &types.AttributeValueMemberS{Value: "123"},
        },
    })

    return err
}

// Operation with timeout
func operationWithTimeout() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
    return err
}

// Operation with deadline
func operationWithDeadline(deadline time.Time) error {
    ctx, cancel := context.WithDeadline(context.Background(), deadline)
    defer cancel()

    _, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
        TableName: aws.String("my-table"),
    })

    return err
}
```

### 4.2 Context Propagation in HTTP Handlers

```go
func httpHandler(w http.ResponseWriter, r *http.Request) {
    // Use request context (includes automatic cancellation on client disconnect)
    ctx := r.Context()

    result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String("bucket"),
        Key:    aws.String("key"),
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer result.Body.Close()

    io.Copy(w, result.Body)
}
```

### 4.3 Context Cancellation Handling

```go
func handleCancellation(ctx context.Context) error {
    _, err := dynamoClient.Query(ctx, &dynamodb.QueryInput{
        TableName: aws.String("my-table"),
        // ... query parameters
    })

    if err != nil {
        // Check if error was due to context cancellation
        if errors.Is(err, context.Canceled) {
            log.Println("Operation was cancelled by caller")
            return fmt.Errorf("operation cancelled: %w", err)
        }
        if errors.Is(err, context.DeadlineExceeded) {
            log.Println("Operation timed out")
            return fmt.Errorf("operation timeout: %w", err)
        }
        return err
    }

    return nil
}
```

---

## 5. Retry and Backoff Patterns

### 5.1 Retry Modes

AWS SDK v2 provides two retry modes:

**Standard Mode (Default):**
- Rate-limited retry attempts
- Exponential backoff with jitter
- Default max attempts: 3

**Adaptive Mode:**
- All features of Standard mode
- Additional attempt rate limiting on throttle responses
- Better for high-throughput scenarios

### 5.2 Configuration

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/aws/retry"
    "github.com/aws/aws-sdk-go-v2/config"
)

func configureRetries(ctx context.Context) (aws.Config, error) {
    return config.LoadDefaultConfig(ctx,
        // Configure retry mode
        config.WithRetryMode(aws.RetryModeAdaptive),

        // Configure max retry attempts
        config.WithRetryMaxAttempts(5),
    )
}
```

### 5.3 Custom Retryer

```go
type customRetryer struct {
    retry.Standard
}

func (r *customRetryer) IsErrorRetryable(err error) bool {
    // Custom retry logic
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        // Retry on specific error codes
        switch apiErr.ErrorCode() {
        case "InternalServerError", "ServiceUnavailable":
            return true
        case "ThrottlingException", "ProvisionedThroughputExceededException":
            return true
        }
    }

    // Fall back to standard retry logic
    return r.Standard.IsErrorRetryable(err)
}

func (r *customRetryer) RetryDelay(attempt int, err error) (time.Duration, error) {
    // Custom backoff: exponential with jitter
    baseDelay := time.Second
    maxDelay := 30 * time.Second

    delay := time.Duration(1<<uint(attempt)) * baseDelay
    if delay > maxDelay {
        delay = maxDelay
    }

    // Add jitter (0-25% of delay)
    jitter := time.Duration(rand.Int63n(int64(delay / 4)))
    return delay + jitter, nil
}

func newConfigWithCustomRetryer(ctx context.Context) (aws.Config, error) {
    return config.LoadDefaultConfig(ctx,
        config.WithRetryer(func() aws.Retryer {
            return &customRetryer{
                Standard: retry.NewStandard(func(so *retry.StandardOptions) {
                    so.MaxAttempts = 5
                }),
            }
        }),
    )
}
```

---

## 6. Testing Patterns

### 6.1 Interface-Based Mocking (Official Pattern)

**Define interfaces:**
```go
type DynamoDBAPI interface {
    GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
    PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type S3API interface {
    GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
    PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}
```

**Use interfaces in production code:**
```go
type UserStore struct {
    client DynamoDBAPI
    table  string
}

func NewUserStore(client DynamoDBAPI, table string) *UserStore {
    return &UserStore{
        client: client,
        table:  table,
    }
}

func (s *UserStore) GetUser(ctx context.Context, id string) (*User, error) {
    result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(s.table),
        Key: map[string]types.AttributeValue{
            "id": &types.AttributeValueMemberS{Value: id},
        },
    })
    // ... parse result
    return user, err
}
```

**Mock in tests:**
```go
type mockDynamoDB struct {
    GetItemFunc func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

func (m *mockDynamoDB) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
    if m.GetItemFunc != nil {
        return m.GetItemFunc(ctx, params, optFns...)
    }
    return nil, fmt.Errorf("GetItemFunc not implemented")
}

func (m *mockDynamoDB) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
    return nil, fmt.Errorf("not implemented")
}

// Test
func TestGetUser(t *testing.T) {
    mock := &mockDynamoDB{
        GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
            return &dynamodb.GetItemOutput{
                Item: map[string]types.AttributeValue{
                    "id":   &types.AttributeValueMemberS{Value: "123"},
                    "name": &types.AttributeValueMemberS{Value: "John Doe"},
                },
            }, nil
        },
    }

    store := NewUserStore(mock, "users")
    user, err := store.GetUser(context.Background(), "123")

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "John Doe" {
        t.Errorf("expected name 'John Doe', got '%s'", user.Name)
    }
}
```

### 6.2 AWS AwsmStubber Pattern

**Setup:**
```go
import (
    "github.com/awsdocs/aws-doc-sdk-examples/gov2/testtools"
)

func TestWithStubber(t *testing.T) {
    // Create stubber
    stubber := testtools.NewStubber()

    // Create service client with stubber config
    client := dynamodb.NewFromConfig(*stubber.SdkConfig)

    // Add expected operation stub
    stubber.Add(testtools.Stub{
        OperationName: "GetItem",
        Input: &dynamodb.GetItemInput{
            TableName: aws.String("users"),
            Key: map[string]types.AttributeValue{
                "id": &types.AttributeValueMemberS{Value: "123"},
            },
        },
        Output: &dynamodb.GetItemOutput{
            Item: map[string]types.AttributeValue{
                "id":   &types.AttributeValueMemberS{Value: "123"},
                "name": &types.AttributeValueMemberS{Value: "John Doe"},
            },
        },
    })

    // Run code under test
    result, err := client.GetItem(context.Background(), &dynamodb.GetItemInput{
        TableName: aws.String("users"),
        Key: map[string]types.AttributeValue{
            "id": &types.AttributeValueMemberS{Value: "123"},
        },
    })

    // Verify
    testtools.VerifyError(err, nil, t)
    if result.Item["name"].(*types.AttributeValueMemberS).Value != "John Doe" {
        t.Errorf("unexpected name")
    }

    // Verify all stubs were called
    testtools.ExitTest(stubber, t)
}
```

**Testing error paths:**
```go
func TestErrorHandling(t *testing.T) {
    stubber := testtools.NewStubber()
    client := dynamodb.NewFromConfig(*stubber.SdkConfig)

    // Stub error response
    stubber.Add(testtools.Stub{
        OperationName: "GetItem",
        Input: &dynamodb.GetItemInput{
            TableName: aws.String("users"),
            Key: map[string]types.AttributeValue{
                "id": &types.AttributeValueMemberS{Value: "999"},
            },
        },
        Error: &types.ResourceNotFoundException{
            Message: aws.String("Item not found"),
        },
    })

    _, err := client.GetItem(context.Background(), &dynamodb.GetItemInput{
        TableName: aws.String("users"),
        Key: map[string]types.AttributeValue{
            "id": &types.AttributeValueMemberS{Value: "999"},
        },
    })

    var notFoundErr *types.ResourceNotFoundException
    if !errors.As(err, &notFoundErr) {
        t.Errorf("expected ResourceNotFoundException, got %v", err)
    }

    testtools.ExitTest(stubber, t)
}
```

### 6.3 Integration Testing with LocalStack/MinIO

```go
// +build integration

func TestS3Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Connect to LocalStack/MinIO
    client, err := newMinIOClient(ctx, "http://localhost:9000")
    if err != nil {
        t.Fatal(err)
    }

    bucketName := "test-bucket-" + uuid.New().String()

    // Create bucket
    _, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String(bucketName),
    })
    if err != nil {
        t.Fatal(err)
    }
    defer func() {
        // Cleanup
        client.DeleteBucket(ctx, &s3.DeleteBucketInput{
            Bucket: aws.String(bucketName),
        })
    }()

    // Test operations
    key := "test-key"
    content := []byte("test content")

    _, err = client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(key),
        Body:   bytes.NewReader(content),
    })
    if err != nil {
        t.Fatal(err)
    }

    result, err := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(key),
    })
    if err != nil {
        t.Fatal(err)
    }
    defer result.Body.Close()

    retrieved, _ := io.ReadAll(result.Body)
    if !bytes.Equal(content, retrieved) {
        t.Errorf("content mismatch")
    }
}
```

---

## 7. Multi-Service Patterns

### 7.1 Service Manager Pattern

```go
type AWSServices struct {
    S3       *s3.Client
    DynamoDB *dynamodb.Client
    SQS      *sqs.Client

    config   aws.Config
}

func NewAWSServices(ctx context.Context, region string) (*AWSServices, error) {
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(region),
    )
    if err != nil {
        return nil, fmt.Errorf("unable to load SDK config: %w", err)
    }

    return &AWSServices{
        S3:       s3.NewFromConfig(cfg),
        DynamoDB: dynamodb.NewFromConfig(cfg),
        SQS:      sqs.NewFromConfig(cfg),
        config:   cfg,
    }, nil
}

// Add new service on demand
func (s *AWSServices) SNS() *sns.Client {
    return sns.NewFromConfig(s.config)
}
```

### 7.2 Factory Pattern

```go
type ServiceFactory struct {
    config aws.Config
    mu     sync.RWMutex

    s3Client      *s3.Client
    dynamoClient  *dynamodb.Client
}

func NewServiceFactory(ctx context.Context) (*ServiceFactory, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, err
    }

    return &ServiceFactory{
        config: cfg,
    }, nil
}

func (f *ServiceFactory) S3() *s3.Client {
    f.mu.RLock()
    if f.s3Client != nil {
        client := f.s3Client
        f.mu.RUnlock()
        return client
    }
    f.mu.RUnlock()

    f.mu.Lock()
    defer f.mu.Unlock()

    // Double-check after acquiring write lock
    if f.s3Client == nil {
        f.s3Client = s3.NewFromConfig(f.config)
    }

    return f.s3Client
}

func (f *ServiceFactory) DynamoDB() *dynamodb.Client {
    f.mu.RLock()
    if f.dynamoClient != nil {
        client := f.dynamoClient
        f.mu.RUnlock()
        return client
    }
    f.mu.RUnlock()

    f.mu.Lock()
    defer f.mu.Unlock()

    if f.dynamoClient == nil {
        f.dynamoClient = dynamodb.NewFromConfig(f.config)
    }

    return f.dynamoClient
}
```

---

## 8. Production Best Practices Summary

### 8.1 Client Lifecycle
- **Create once, reuse everywhere** - SDK clients are thread-safe
- **No Close() method needed** - Clients are garbage collected
- **Share configuration** - Use same `aws.Config` for multiple services
- **Singleton or factory patterns** - For application-wide client access

### 8.2 Error Handling
- **Always check errors** - SDK operations can fail for many reasons
- **Use errors.As for typed errors** - Extract specific error types
- **Log with context** - Include operation details in error messages
- **Wrap errors** - Add context with `fmt.Errorf("context: %w", err)`

### 8.3 Resource Management
- **Always defer response body Close()** - For S3 GetObject operations
- **Always defer file Close()** - When working with files
- **Always defer context cancel()** - Prevent context leaks
- **No need to close SDK clients** - They manage their own resources

### 8.4 Context Usage
- **Use context.Background() for long-lived operations**
- **Use context.WithTimeout() for bounded operations**
- **Propagate request context in HTTP handlers**
- **Check for context.Canceled and context.DeadlineExceeded**

### 8.5 HTTP Client Configuration
- **Tune connection pooling** - Based on workload (default: 100 total, 10 per host)
- **Set appropriate timeouts** - Default 30s may be too long/short
- **Configure keep-alive** - For connection reuse
- **Share HTTP client** - Across all SDK clients for efficiency

### 8.6 Retry Configuration
- **Use adaptive mode for high throughput** - Better throttle handling
- **Set appropriate max attempts** - Default is 3, increase for critical operations
- **Implement custom retry logic** - For application-specific retry needs
- **Monitor retry metrics** - Track retry rates in production

### 8.7 Testing
- **Use interface-based mocking** - For unit tests
- **Use AwsmStubber for complex scenarios** - Official testing framework
- **Integration tests with LocalStack/MinIO** - For realistic testing
- **Tag integration tests** - Use build tags to separate unit/integration

### 8.8 Security
- **Never hardcode credentials** - Use IAM roles, environment vars, or config files
- **Use temporary credentials** - STS for cross-account access
- **Validate inputs** - Before sending to AWS
- **Handle secrets carefully** - Use AWS Secrets Manager or Parameter Store

---

## 9. Common Pitfalls to Avoid

### 9.1 Resource Leaks
```go
// BAD - Response body not closed
result, _ := s3Client.GetObject(ctx, input)
data, _ := io.ReadAll(result.Body)  // Leak!

// GOOD
result, err := s3Client.GetObject(ctx, input)
if err != nil {
    return err
}
defer result.Body.Close()  // Always close
data, err := io.ReadAll(result.Body)
```

### 9.2 Creating Clients Repeatedly
```go
// BAD - Creates new client for each request
func handleRequest(ctx context.Context) error {
    cfg, _ := config.LoadDefaultConfig(ctx)  // Expensive!
    client := s3.NewFromConfig(cfg)          // Expensive!
    // use client...
}

// GOOD - Create once, reuse
var s3Client *s3.Client

func init() {
    cfg, _ := config.LoadDefaultConfig(context.Background())
    s3Client = s3.NewFromConfig(cfg)
}

func handleRequest(ctx context.Context) error {
    // use s3Client...
}
```

### 9.3 Ignoring Context Cancellation
```go
// BAD - Context timeout ignored
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

for i := 0; i < 100; i++ {
    // Each operation takes 500ms, will exceed timeout
    client.PutItem(ctx, input)  // Later calls will fail
}

// GOOD - Check context before operations
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

for i := 0; i < 100; i++ {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        if err := client.PutItem(ctx, input); err != nil {
            return err
        }
    }
}
```

### 9.4 Not Using Path-Style for MinIO
```go
// BAD - Virtual-hosted style doesn't work with MinIO
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String("http://localhost:9000")
    // Missing: o.UsePathStyle = true
})

// GOOD
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String("http://localhost:9000")
    o.UsePathStyle = true  // Required for MinIO
})
```

### 9.5 Improper Error Type Checking
```go
// BAD - String comparison is fragile
if err != nil && strings.Contains(err.Error(), "NotFound") {
    // Fragile!
}

// GOOD - Use typed errors
var notFound *types.ResourceNotFoundException
if errors.As(err, &notFound) {
    // Robust!
}
```

---

## 10. Production Configuration Example

Complete production-ready configuration:

```go
package aws

import (
    "context"
    "fmt"
    "log"
    "net"
    "net/http"
    "sync"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/aws/retry"
    awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
    Region              string
    MaxRetries          int
    RequestTimeout      time.Duration
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    IdleConnTimeout     time.Duration
}

type Clients struct {
    S3       *s3.Client
    DynamoDB *dynamodb.Client

    cfg    aws.Config
    once   sync.Once
    initErr error
}

var (
    instance *Clients
    once     sync.Once
)

func NewClients(ctx context.Context, cfg *Config) (*Clients, error) {
    if cfg == nil {
        cfg = &Config{
            Region:              "us-west-2",
            MaxRetries:          5,
            RequestTimeout:      30 * time.Second,
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        }
    }

    // Configure HTTP client
    httpClient := awshttp.NewBuildableClient().
        WithTimeout(cfg.RequestTimeout).
        WithTransportOptions(func(tr *http.Transport) {
            tr.MaxIdleConns = cfg.MaxIdleConns
            tr.MaxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
            tr.IdleConnTimeout = cfg.IdleConnTimeout
            tr.TLSHandshakeTimeout = 10 * time.Second
            tr.ResponseHeaderTimeout = 10 * time.Second
            tr.ExpectContinueTimeout = 1 * time.Second

            tr.DialContext = (&net.Dialer{
                Timeout:   30 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext
        })

    // Load AWS configuration
    awsCfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(cfg.Region),
        config.WithHTTPClient(httpClient),
        config.WithRetryMode(aws.RetryModeAdaptive),
        config.WithRetryMaxAttempts(cfg.MaxRetries),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    clients := &Clients{
        cfg: awsCfg,
    }

    // Initialize clients
    clients.S3 = s3.NewFromConfig(awsCfg)
    clients.DynamoDB = dynamodb.NewFromConfig(awsCfg)

    log.Printf("AWS clients initialized successfully for region: %s", cfg.Region)

    return clients, nil
}

// GetInstance returns singleton instance
func GetInstance(ctx context.Context) (*Clients, error) {
    once.Do(func() {
        instance, _ = NewClients(ctx, nil)
    })

    if instance == nil {
        return nil, fmt.Errorf("failed to initialize AWS clients")
    }

    return instance, nil
}
```

---

## 11. References

### Official Documentation
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- [Developer Guide](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/welcome.html)
- [Migration Guide from v1](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/migrate-gosdk.html)
- [Code Examples](https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2)

### Key Packages
- `github.com/aws/aws-sdk-go-v2/config` - Configuration loading
- `github.com/aws/aws-sdk-go-v2/aws` - Core types and interfaces
- `github.com/aws/aws-sdk-go-v2/credentials` - Credential providers
- `github.com/aws/aws-sdk-go-v2/aws/retry` - Retry configuration
- `github.com/aws/smithy-go` - Error handling

### Testing
- [Unit Testing Guide](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/unit-testing.html)
- [testtools Package](https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/testtools)

### Community Resources
- [HashiCorp aws-sdk-go-base](https://github.com/hashicorp/aws-sdk-go-base)
- [MinIO AWS SDK Integration](https://github.com/minio/minio/discussions)
- [Terraform AWS Provider](https://github.com/hashicorp/terraform-provider-aws)

---

## Conclusion

AWS SDK for Go v2 provides a modern, production-ready framework for AWS service interactions. Key takeaways:

1. **Clients are thread-safe and reusable** - Create once, use everywhere
2. **No Close() methods** - SDK manages resources automatically
3. **Always close response bodies** - For S3 GetObject operations
4. **Use context properly** - For timeouts, cancellation, and propagation
5. **Handle errors with errors.As** - Extract typed errors reliably
6. **Configure HTTP client** - Tune for your workload
7. **Test with interfaces** - Enable easy mocking
8. **Use adaptive retry mode** - For production resilience

This analysis is based on official AWS documentation, production repositories (HashiCorp, Terraform), and community best practices as of 2024-2025.
