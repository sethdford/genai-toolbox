# CloudWatch Logs Source

A comprehensive AWS CloudWatch Logs source integration for the GenAI Toolbox. This source provides access to AWS CloudWatch Logs for querying, streaming, and analyzing log data.

## Features

- **FilterLogEvents API**: Simple log filtering and streaming
- **CloudWatch Logs Insights**: Complex analytical queries over historical log data
- **Log Discovery**: List log groups and log streams
- **Flexible Authentication**: Supports AWS credential chain and explicit credentials
- **LocalStack Support**: Custom endpoint support for local development
- **Comprehensive Error Handling**: Detailed error messages and proper AWS SDK v2 patterns

## Configuration

### Basic Configuration

```yaml
name: my-cloudwatch
kind: cloudwatch
region: us-east-1
```

### With Default Log Group

```yaml
name: my-cloudwatch
kind: cloudwatch
region: us-west-2
logGroupName: /aws/lambda/my-function
```

### With Explicit Credentials

```yaml
name: my-cloudwatch
kind: cloudwatch
region: eu-west-1
accessKeyId: AKIAIOSFODNN7EXAMPLE
secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### With Session Token (for temporary credentials)

```yaml
name: my-cloudwatch
kind: cloudwatch
region: ap-southeast-1
accessKeyId: AKIAIOSFODNN7EXAMPLE
secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
sessionToken: FwoGZXIvYXdzEBQaDH1234567890EXAMPLE
```

### LocalStack (for local testing)

```yaml
name: my-cloudwatch-local
kind: cloudwatch
region: us-east-1
endpoint: http://localhost:4566
logGroupName: /aws/lambda/test
```

## Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique name for this source |
| `kind` | string | Yes | Must be "cloudwatch" |
| `region` | string | Yes | AWS region (e.g., us-east-1) |
| `logGroupName` | string | No | Default log group for queries |
| `endpoint` | string | No | Custom endpoint (for LocalStack) |
| `accessKeyId` | string | No | AWS access key ID |
| `secretAccessKey` | string | No | AWS secret access key |
| `sessionToken` | string | No | AWS session token (for temporary credentials) |

## Usage Examples

### Filtering Log Events

Simple log filtering is suitable for real-time log streaming and basic queries:

```go
ctx := context.Background()

// Create input for filtering
input := &cloudwatch.FilterLogEventsInput{
    LogGroupName: "/aws/lambda/my-function",
    StartTime: time.Now().Add(-1 * time.Hour),
    EndTime: time.Now(),
    FilterPattern: "[level=ERROR]",
    Limit: 100,
}

// Filter log events
output, err := source.FilterLogEvents(ctx, input)
if err != nil {
    log.Fatalf("Failed to filter logs: %v", err)
}

// Process events
for _, event := range output.Events {
    fmt.Printf("[%s] %s: %s\n",
        time.UnixMilli(event.Timestamp).Format(time.RFC3339),
        event.LogStreamName,
        event.Message)
}

// Handle pagination
if output.NextToken != nil {
    input.NextToken = *output.NextToken
    // Make another call to get more events
}
```

### CloudWatch Logs Insights Query

For complex analytical queries over historical data:

```go
ctx := context.Background()

// Start an Insights query
queryInput := &cloudwatch.InsightsQueryInput{
    LogGroupNames: []string{"/aws/lambda/my-function"},
    QueryString: `
        fields @timestamp, @message, @requestId
        | filter @message like /ERROR/
        | stats count() as error_count by bin(5m)
        | sort @timestamp desc
        | limit 100
    `,
    StartTime: time.Now().Add(-24 * time.Hour),
    EndTime: time.Now(),
    Limit: 1000,
}

queryOutput, err := source.StartInsightsQuery(ctx, queryInput)
if err != nil {
    log.Fatalf("Failed to start query: %v", err)
}

// Poll for results
for {
    results, err := source.GetInsightsQueryResults(ctx, queryOutput.QueryID)
    if err != nil {
        log.Fatalf("Failed to get results: %v", err)
    }

    switch results.Status {
    case "Complete":
        // Process results
        for _, row := range results.Results {
            for _, field := range row {
                fmt.Printf("%s: %s\n", field.Field, field.Value)
            }
            fmt.Println("---")
        }

        // Print statistics
        if results.Statistics != nil {
            fmt.Printf("Bytes scanned: %.0f\n", results.Statistics.BytesScanned)
            fmt.Printf("Records matched: %.0f\n", results.Statistics.RecordsMatched)
            fmt.Printf("Records scanned: %.0f\n", results.Statistics.RecordsScanned)
        }
        return

    case "Failed", "Cancelled":
        log.Fatalf("Query %s: %s", results.Status, queryOutput.QueryID)

    case "Running", "Scheduled":
        time.Sleep(1 * time.Second)
        continue
    }
}
```

### Listing Log Groups

Discover available log groups:

```go
ctx := context.Background()

logGroups, nextToken, err := source.ListLogGroups(ctx, 50, "")
if err != nil {
    log.Fatalf("Failed to list log groups: %v", err)
}

fmt.Println("Available log groups:")
for _, lg := range logGroups {
    fmt.Println(lg)
}

// Handle pagination
if nextToken != "" {
    moreGroups, _, _ := source.ListLogGroups(ctx, 50, nextToken)
    // Process more groups...
}
```

### Listing Log Streams

Discover log streams in a log group:

```go
ctx := context.Background()

streams, nextToken, err := source.ListLogStreams(ctx, "/aws/lambda/my-function", 50, "")
if err != nil {
    log.Fatalf("Failed to list log streams: %v", err)
}

fmt.Println("Available log streams:")
for _, stream := range streams {
    if stream.LogStreamName != nil {
        fmt.Printf("- %s (Last event: %v)\n",
            *stream.LogStreamName,
            stream.LastEventTime)
    }
}
```

## CloudWatch Logs Insights Query Language

The Insights query language supports:

- **Field selection**: `fields @timestamp, @message, @requestId`
- **Filtering**: `filter @message like /ERROR/` or `filter statusCode >= 400`
- **Statistics**: `stats count(), avg(duration), max(bytes) by bin(5m)`
- **Sorting**: `sort @timestamp desc`
- **Limiting**: `limit 100`
- **Parsing**: `parse @message /User: (?<user>.*)/`
- **Regular expressions**: Pattern matching and extraction

### Example Queries

**Count errors by time:**
```
fields @timestamp, @message
| filter @message like /ERROR/
| stats count() as error_count by bin(5m)
```

**Find slow requests:**
```
fields @timestamp, @requestId, duration
| filter duration > 1000
| sort duration desc
| limit 20
```

**Parse custom log format:**
```
fields @timestamp, @message
| parse @message /User: (?<user>.*) Action: (?<action>.*)/
| stats count() by user, action
```

**Aggregate by multiple dimensions:**
```
fields @timestamp
| stats count() as total,
        avg(duration) as avg_duration,
        max(bytes) as max_bytes
  by bin(1h), statusCode
| sort total desc
```

## Filter Patterns

CloudWatch Logs filter patterns support:

- **Simple text**: `ERROR` - matches logs containing "ERROR"
- **JSON fields**: `{ $.level = "ERROR" }` - matches JSON logs
- **Metric filters**: `[level=ERROR]` - extracts structured data
- **Multiple terms**: `[time, request_id, event_type = "error"]`
- **Wildcards**: `[..., status_code = 4*, ...]`

### Example Filter Patterns

```
[level=ERROR]                    # Match ERROR level
{ $.statusCode >= 400 }          # JSON with status code >= 400
[..., "Error", ...]              # Contains "Error" anywhere
[ip, user, timestamp, request]   # Structured log format
```

## Authentication

The source supports multiple authentication methods (in order of precedence):

1. **Explicit credentials** in configuration (accessKeyId, secretAccessKey, sessionToken)
2. **Environment variables** (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
3. **Shared credentials file** (~/.aws/credentials)
4. **IAM role** (when running on EC2, ECS, Lambda, etc.)

## AWS Permissions Required

The IAM user or role needs these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
        "logs:FilterLogEvents",
        "logs:StartQuery",
        "logs:GetQueryResults"
      ],
      "Resource": "*"
    }
  ]
}
```

For specific log groups, restrict the resource:

```json
{
  "Resource": "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/*"
}
```

## Best Practices

### Performance

1. **Use time ranges**: Always specify StartTime and EndTime to limit data scanned
2. **Limit results**: Use the Limit parameter to control result size
3. **Use Insights for complex queries**: FilterLogEvents is better for simple filtering
4. **Paginate results**: Use NextToken for large result sets

### Cost Optimization

1. **Narrow time ranges**: CloudWatch Logs charges by data scanned
2. **Use filter patterns**: Filter early to reduce data transferred
3. **Monitor query costs**: Check Statistics.BytesScanned
4. **Cache results**: Store frequently accessed query results

### Query Optimization

1. **Filter before aggregating**: `filter` before `stats` reduces data processed
2. **Use bins wisely**: Larger bins (e.g., 1h vs 1m) process less data
3. **Limit field selection**: Select only needed fields
4. **Index on timestamp**: Queries filtering by time are faster

## Error Handling

Common errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| `ResourceNotFoundException` | Log group doesn't exist | Verify log group name |
| `InvalidParameterException` | Invalid query syntax | Check query language syntax |
| `ThrottlingException` | Too many requests | Implement exponential backoff |
| `LimitExceededException` | Query too large | Reduce time range or add filters |
| `ServiceUnavailableException` | Temporary service issue | Retry with backoff |

## Testing

### Unit Tests

```bash
go test ./internal/sources/cloudwatch/...
```

### Integration Tests with LocalStack

```bash
# Start LocalStack
docker run -d -p 4566:4566 localstack/localstack

# Configure LocalStack endpoint
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Run tests
go test ./internal/sources/cloudwatch/... -tags=integration
```

## Architecture

The CloudWatch source follows AWS SDK v2 patterns:

- **No Close() method**: AWS SDK v2 clients manage their own lifecycle
- **Context-based cancellation**: All operations accept context.Context
- **Proper error wrapping**: Errors include context about the operation
- **Credential providers**: Flexible authentication via AWS credential chain
- **Region configuration**: Required for all operations
- **Custom endpoints**: Support for LocalStack and custom endpoints

## Limitations

1. **Query timeout**: Insights queries have a 15-minute timeout
2. **Result size**: Maximum 10,000 rows per query
3. **Time range**: Maximum 366 days (queries spanning archived logs)
4. **Concurrent queries**: Limited by AWS quotas (default: 30 concurrent)
5. **Data scanned**: Charged per GB scanned (see AWS pricing)

## References

- [CloudWatch Logs Documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/)
- [CloudWatch Logs Insights Query Syntax](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html)
- [AWS SDK for Go v2 - CloudWatch Logs](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs)
- [CloudWatch Logs Pricing](https://aws.amazon.com/cloudwatch/pricing/)
