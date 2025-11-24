# AWS SDK Go v2 - Quick Reference Card

## Essential Patterns

### 1. Basic Configuration
```go
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),  // REQUIRED - no default region
)
```

### 2. Static Credentials
```go
creds := credentials.NewStaticCredentialsProvider(
    "ACCESS_KEY",
    "SECRET_KEY",
    "",  // Session token (empty if not using)
)
cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithCredentialsProvider(creds),
)
```

### 3. Assume Role
```go
cfg, _ := config.LoadDefaultConfig(context.TODO())
stsClient := sts.NewFromConfig(cfg)
roleProvider := stscreds.NewAssumeRoleProvider(stsClient, "role-arn")
cfg.Credentials = aws.NewCredentialsCache(roleProvider)
```

### 4. Service Client Creation
```go
// Create once, reuse everywhere (thread-safe)
client := s3.NewFromConfig(cfg)
```

### 5. Custom Endpoint (DynamoDB Local, LocalStack)
```go
endpoint := "http://localhost:8000"
client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
    o.BaseEndpoint = &endpoint
})
```

### 6. S3 Path-Style Addressing
```go
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.UsePathStyle = true
})
```

### 7. Retry Configuration
```go
cfg, _ := config.LoadDefaultConfig(context.TODO(),
    config.WithRetryer(func() aws.Retryer {
        return retry.AddWithMaxAttempts(retry.NewStandard(), 5)
    }),
)
```

### 8. Timeout with Context
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := client.GetObject(ctx, &s3.GetObjectInput{...})
```

### 9. ALWAYS Close Response Bodies
```go
resp, err := client.GetObject(ctx, input)
if err != nil {
    return err
}
defer resp.Body.Close()  // CRITICAL
data, _ := io.ReadAll(resp.Body)
```

### 10. Error Handling
```go
import "errors"

_, err := client.GetObject(ctx, input)
if err != nil {
    var nsk *types.NoSuchKey
    if errors.As(err, &nsk) {
        // Handle specific error
    }
}
```

## Credential Chain Order

1. Programmatic options (highest)
2. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
3. `~/.aws/credentials` and `~/.aws/config`
4. IAM role for ECS tasks
5. IAM role for EC2 instances

## Common Gotchas

| Issue | Fix |
|-------|-----|
| No default region | Always specify with `config.WithRegion()` |
| Connection leaks | ALWAYS `defer resp.Body.Close()` |
| Creating clients in loops | Create once at startup, reuse |
| Infinite retries | Set max attempts |
| String error comparison | Use `errors.As()` |

## Service-Specific Notes

### DynamoDB Local
- Endpoint: `http://localhost:8000`
- Still requires region setting
- Dummy credentials work

### S3
- Default: Virtual-hosted-style
- Use `UsePathStyle: true` for LocalStack/MinIO
- Directory buckets require virtual-hosted-style

### Athena
1. `StartQueryExecution` - Get query ID
2. `GetQueryExecution` - Poll status
3. `GetQueryResults` - Get results

### Timestream
- Needs TWO clients: `timestreamwrite` and `timestreamquery`
- Write client for ingestion
- Query client for retrieval

## Connection Management

- Clients DON'T need Close() - reuse them
- Response bodies DO need Close() - always defer
- Connection pooling is automatic
- Default: 100 max idle, 10 per host

## Must-Know Rules

1. **No default region** - Always specify
2. **Close all io.ReadCloser** - Prevents leaks
3. **Clients are thread-safe** - Share them
4. **Create clients once** - Don't recreate
5. **Use context for timeouts** - All operations accept it

## Full Documentation

See `AWS_SDK_GO_V2_PATTERNS.md` for complete reference with examples and links.
