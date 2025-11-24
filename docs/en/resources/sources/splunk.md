---
title: "Splunk"
linkTitle: "Splunk"
type: docs
weight: 1
description: >
  The Splunk source enables the Toolbox to connect to Splunk for searching logs, metrics, and sending events via HTTP Event Collector (HEC).
---

## About

The Splunk Source allows Toolbox to interact with Splunk Enterprise or Splunk Cloud instances. This enables Generative AI applications to search and analyze log data, metrics, and machine data stored in Splunk, as well as send events to Splunk using the HTTP Event Collector (HEC).

## Features

- **Dual Authentication**: Supports both token-based and username/password authentication
- **Search API**: Create and manage search jobs with full SPL (Search Processing Language) support
- **HTTP Event Collector**: Send structured and raw events to Splunk
- **Flexible Configuration**: Customizable ports, schemes, and timeouts
- **SSL/TLS Support**: Configurable SSL verification for secure connections

## Available Operations

The Splunk source provides the following capabilities:

- **Search Operations**
  - Create search jobs with SPL queries
  - Monitor search job status
  - Retrieve search results with pagination
  - Delete completed search jobs

- **Event Collection**
  - Send JSON-formatted events via HEC
  - Send raw events via HEC
  - Support for custom fields, source, sourcetype, and index

## Example Configurations

### Token-Based Authentication

```yaml
sources:
  my-splunk-prod:
    kind: splunk
    host: splunk.example.com
    token: ${SPLUNK_TOKEN}
    hecToken: ${SPLUNK_HEC_TOKEN}  # Optional: for HEC operations
    timeout: 120s  # default to 120s
```

### Username/Password Authentication

```yaml
sources:
  my-splunk-dev:
    kind: splunk
    host: splunk-dev.example.com
    username: ${SPLUNK_USERNAME}
    password: ${SPLUNK_PASSWORD}
    port: 8089
    hecPort: 8088
```

### Development Configuration (HTTP)

```yaml
sources:
  splunk-local:
    kind: splunk
    host: localhost
    scheme: http
    port: 8089
    hecPort: 8088
    token: dev-token-12345
    disableSslVerification: true  # Only for local development
```

### Advanced Configuration

```yaml
sources:
  splunk-advanced:
    kind: splunk
    host: splunk.cloud.example.com
    port: 8089
    hecPort: 8088
    scheme: https
    token: ${SPLUNK_API_TOKEN}
    hecToken: ${SPLUNK_HEC_TOKEN}
    timeout: 300s
    disableSslVerification: false
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

{{< notice warning >}}
The `disableSslVerification` option should only be used for local development
or testing environments. Always use SSL verification in production.
{{< /notice >}}

## Authentication Methods

### Token Authentication (Recommended)

Token-based authentication uses Splunk authentication tokens (JWT) for API access:

```yaml
sources:
  my-splunk:
    kind: splunk
    host: splunk.example.com
    token: B5A79AAD-D822-46CC-80D1-819F80D7BFB0
```

To create a token in Splunk:
1. Navigate to Settings > Tokens
2. Click "New Token"
3. Configure expiration and permissions
4. Copy the generated token

### Username/Password Authentication

Uses standard Splunk credentials to obtain a session key:

```yaml
sources:
  my-splunk:
    kind: splunk
    host: splunk.example.com
    username: admin
    password: ${SPLUNK_PASSWORD}
```

The source automatically authenticates and manages the session key.

## HTTP Event Collector (HEC)

To use HEC for sending events, configure the `hecToken` field:

```yaml
sources:
  my-splunk:
    kind: splunk
    host: splunk.example.com
    token: ${SPLUNK_API_TOKEN}
    hecToken: ${SPLUNK_HEC_TOKEN}
    hecPort: 8088  # default
```

HEC tokens are separate from API tokens and must be created in Splunk:
1. Navigate to Settings > Data Inputs > HTTP Event Collector
2. Click "New Token"
3. Configure source, sourcetype, and index settings
4. Copy the generated HEC token

## Usage Examples

### Creating a Search Job

```go
// Create a search job
params := map[string]string{
    "earliest_time": "-1h",
    "latest_time": "now",
}
jobResp, err := splunkSource.CreateSearchJob(ctx,
    "search index=main error | head 100",
    params)
if err != nil {
    return err
}
sid := jobResp.SID

// Check job status
status, err := splunkSource.GetSearchJobStatus(ctx, sid)
if err != nil {
    return err
}

// Wait for completion and get results
if status.Entry[0].Content.IsDone {
    results, err := splunkSource.GetSearchResults(ctx, sid, 0, 100)
    if err != nil {
        return err
    }
    // Process results...
}

// Clean up
err = splunkSource.DeleteSearchJob(ctx, sid)
```

### Sending Events via HEC

```go
// Send a structured event
event := &splunk.HECEvent{
    Event: map[string]interface{}{
        "message": "User login",
        "severity": "info",
        "user": "john.doe",
    },
    Source: "myapp",
    SourceType: "application:log",
    Index: "main",
    Fields: map[string]interface{}{
        "environment": "production",
        "region": "us-west-2",
    },
}
err := splunkSource.SendHECEvent(ctx, event)

// Send a raw event
rawEvent := "2025-01-15 10:30:00 INFO User login successful"
params := map[string]string{
    "sourcetype": "application:log",
    "index": "main",
}
err := splunkSource.SendHECRawEvent(ctx, rawEvent, params)
```

## Reference

| **field**              |  **type** | **required** | **description**                                                                                                                        |
|------------------------|:---------:|:------------:|----------------------------------------------------------------------------------------------------------------------------------------|
| kind                   |  string   |     true     | Must be "splunk".                                                                                                                      |
| host                   |  string   |     true     | The Splunk server hostname or IP address (e.g., `splunk.example.com`).                                                                 |
| port                   |    int    |    false     | The Splunk management port for REST API. Defaults to `8089`.                                                                            |
| hecPort                |    int    |    false     | The HTTP Event Collector port. Defaults to `8088`.                                                                                      |
| scheme                 |  string   |    false     | The connection scheme (`http` or `https`). Defaults to `https`.                                                                         |
| token                  |  string   |    false     | Splunk authentication token for REST API access. Required if username/password not provided.                                            |
| username               |  string   |    false     | Splunk username for authentication. Required if token not provided.                                                                      |
| password               |  string   |    false     | Splunk password for authentication. Required if token not provided.                                                                      |
| hecToken               |  string   |    false     | HTTP Event Collector token for sending events. Required only for HEC operations.                                                        |
| timeout                |  string   |    false     | The timeout for HTTP requests (e.g., "120s", "5m", refer to [ParseDuration][parse-duration-doc]). Defaults to `120s`.                   |
| disableSslVerification |   bool    |    false     | Disable SSL certificate verification. This should only be used for local development. Defaults to `false`.                              |

[parse-duration-doc]: https://pkg.go.dev/time#ParseDuration

## API Endpoints

The Splunk source uses the following REST API endpoints:

### Authentication
- `POST /services/auth/login` - Obtain session key (username/password auth)
- `GET /services/server/info` - Test connection

### Search API
- `POST /services/search/jobs` - Create search job
- `GET /services/search/jobs/{sid}` - Get search job status
- `GET /services/search/jobs/{sid}/results` - Get search results
- `DELETE /services/search/jobs/{sid}` - Delete search job

### HTTP Event Collector
- `POST /services/collector/event` - Send JSON event
- `POST /services/collector/raw` - Send raw event

## Best Practices

1. **Use Token Authentication**: Token-based authentication is more secure and easier to manage than username/password.

2. **Set Appropriate Timeouts**: Adjust the `timeout` value based on your search complexity. Long-running searches may need higher timeout values.

3. **Manage Search Jobs**: Always delete completed search jobs to free up Splunk resources:
   ```go
   defer splunkSource.DeleteSearchJob(ctx, sid)
   ```

4. **Use Environment Variables**: Store sensitive credentials in environment variables:
   ```yaml
   token: ${SPLUNK_TOKEN}
   password: ${SPLUNK_PASSWORD}
   ```

5. **Enable SSL in Production**: Never disable SSL verification in production environments.

6. **Separate HEC Tokens**: Use different HEC tokens for different applications to enable better access control and monitoring.

7. **Pagination for Large Results**: Use the `offset` and `count` parameters when retrieving large search results:
   ```go
   results, err := splunkSource.GetSearchResults(ctx, sid, offset, 1000)
   ```

## Troubleshooting

### Connection Errors

If you receive connection errors, verify:
- The Splunk host is accessible from your network
- The port numbers are correct (default: 8089 for API, 8088 for HEC)
- SSL certificates are valid (or `disableSslVerification: true` for development)

### Authentication Errors

For authentication failures:
- Verify token is valid and not expired
- Check username/password credentials
- Ensure the user has sufficient permissions
- For HEC, verify the HEC token is enabled and valid

### Search Job Errors

If search jobs fail:
- Check SPL syntax is valid
- Verify user has access to the specified indexes
- Ensure time ranges are valid
- Check Splunk search quota limits

## Security Considerations

1. **Token Rotation**: Regularly rotate authentication tokens and HEC tokens
2. **Least Privilege**: Grant minimum required permissions to API tokens
3. **Network Security**: Use HTTPS in production and consider network firewalls
4. **Credential Storage**: Never commit credentials to version control
5. **Audit Logging**: Enable audit logging in Splunk to track API usage

## Performance Tips

1. **Optimize SPL Queries**: Use efficient SPL to reduce search times
2. **Time Range Limits**: Specify appropriate time ranges to limit data volume
3. **Result Pagination**: Retrieve results in batches using pagination
4. **Connection Pooling**: The source reuses HTTP connections for better performance
5. **Batch HEC Events**: Send multiple events in batches when possible

## Links

- [Splunk REST API Documentation](https://docs.splunk.com/Documentation/Splunk/latest/RESTREF/RESTprolog)
- [HTTP Event Collector Documentation](https://docs.splunk.com/Documentation/Splunk/latest/Data/UsetheHTTPEventCollector)
- [Splunk Search Reference](https://docs.splunk.com/Documentation/Splunk/latest/SearchReference/)
- [Authentication Tokens](https://docs.splunk.com/Documentation/SplunkCloud/latest/Security/CreateAuthTokens)
