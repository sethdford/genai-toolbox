# Tableau REST API Go Client

Complete, production-ready Go client for Tableau REST API authentication and operations.

## Quick Start

### Installation

```bash
# Copy the client file to your project
cp tableau_client.go /path/to/your/project/tableau/

# Or add to your Go module
go mod init yourproject
```

### Environment Setup

Create a `.env` file or set environment variables:

```bash
# Required
export TABLEAU_SERVER="https://tableau.example.com"
export TABLEAU_PAT_NAME="your-token-name"
export TABLEAU_PAT_SECRET="your-token-secret"

# Optional
export TABLEAU_SITE=""  # Empty for default site, or site content URL
export TABLEAU_API_VERSION="3.27"
```

### Basic Usage

```go
package main

import (
    "log"
    "os"
    "time"
    "yourproject/tableau"
)

func main() {
    // Create client
    client := tableau.NewClient(tableau.ClientConfig{
        ServerURL:  os.Getenv("TABLEAU_SERVER"),
        APIVersion: "3.27",
        Timeout:    30 * time.Second,
    })

    // Authenticate with PAT (recommended)
    err := client.SignInWithPAT(
        os.Getenv("TABLEAU_PAT_NAME"),
        os.Getenv("TABLEAU_PAT_SECRET"),
        os.Getenv("TABLEAU_SITE"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.SignOut()

    log.Printf("Authenticated! Site ID: %s", client.SiteID)

    // Make API requests
    resp, err := client.DoRequest("GET", "/sites/"+client.SiteID+"/workbooks", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    log.Printf("Response status: %d", resp.StatusCode)
}
```

## Features

- ✅ **Personal Access Token (PAT) Authentication** - Most secure method
- ✅ **Username/Password Authentication** - For development/testing
- ✅ **Automatic Token Management** - Track expiry and refresh
- ✅ **Comprehensive Error Handling** - Detailed error codes and messages
- ✅ **Production-Ready** - Connection pooling, timeouts, retry logic
- ✅ **Well-Tested** - Complete test coverage included
- ✅ **Latest API Support** - API version 3.27 (Tableau 2025.3)

## API Documentation

### Client Methods

#### Authentication

```go
// Sign in with Personal Access Token (recommended)
err := client.SignInWithPAT(tokenName, tokenSecret, siteContentUrl)

// Sign in with username/password
err := client.SignInWithPassword(username, password, siteContentUrl)

// Sign out (invalidates token)
err := client.SignOut()

// Check if authenticated
isAuth := client.IsAuthenticated()

// Check if token is expired or near expiry
isExpired := client.IsTokenExpired()

// Ensure authenticated (refresh if needed)
err := client.EnsureAuthenticated(credentials)
```

#### Making Requests

```go
// Make authenticated API request
resp, err := client.DoRequest(method, endpoint, body)

// Example: GET request
resp, err := client.DoRequest("GET", "/sites/"+client.SiteID+"/workbooks", nil)

// Example: POST request with body
jsonBody, _ := json.Marshal(data)
resp, err := client.DoRequest("POST", "/sites/"+client.SiteID+"/workbooks", jsonBody)
```

### Client Properties

```go
client.Token       // Authentication token
client.SiteID      // Current site ID
client.UserID      // Authenticated user ID
client.TokenExpiry // When the token expires
```

### Error Handling

```go
err := client.SignInWithPAT(tokenName, tokenSecret, site)
if err != nil {
    // Check error type
    if tableau.IsAuthError(err) {
        log.Println("Authentication error")
    }

    // Get detailed error information
    if tableauErr, ok := err.(*tableau.TableauError); ok {
        log.Printf("Error Code: %s", tableauErr.ErrorCode)
        log.Printf("Summary: %s", tableauErr.Summary)
        log.Printf("Detail: %s", tableauErr.Detail)
        log.Printf("HTTP Status: %d", tableauErr.StatusCode)
    }
}
```

## Common Error Codes

| Code   | Description                      |
|--------|----------------------------------|
| 401000 | No authentication credentials    |
| 401001 | Login error / invalid PAT        |
| 401002 | Invalid authentication credentials|
| 401003 | Switch site error                |
| 403000 | Forbidden (wrong site)           |
| 404000 | Resource not found               |

## Examples

See `tableau_example.go` for 9 complete examples:

1. **Basic PAT Authentication**
2. **Username/Password Authentication**
3. **Error Handling**
4. **Making API Requests**
5. **Automatic Token Refresh**
6. **Production-Ready Client**
7. **Retry Logic with Exponential Backoff**
8. **Concurrent Requests**
9. **Complete Application Template**

### Example: List All Workbooks

```go
package main

import (
    "encoding/json"
    "io"
    "log"
    "os"
    "yourproject/tableau"
)

type WorkbooksResponse struct {
    Workbooks struct {
        Workbook []struct {
            ID          string `json:"id"`
            Name        string `json:"name"`
            Description string `json:"description"`
        } `json:"workbook"`
    } `json:"workbooks"`
}

func main() {
    client := tableau.NewClient(tableau.ClientConfig{
        ServerURL:  os.Getenv("TABLEAU_SERVER"),
        APIVersion: "3.27",
    })

    err := client.SignInWithPAT(
        os.Getenv("TABLEAU_PAT_NAME"),
        os.Getenv("TABLEAU_PAT_SECRET"),
        os.Getenv("TABLEAU_SITE"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.SignOut()

    // Fetch workbooks
    endpoint := "/sites/" + client.SiteID + "/workbooks"
    resp, err := client.DoRequest("GET", endpoint, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var workbooks WorkbooksResponse
    if err := json.Unmarshal(body, &workbooks); err != nil {
        log.Fatal(err)
    }

    // Print workbooks
    for _, wb := range workbooks.Workbooks.Workbook {
        log.Printf("Workbook: %s (ID: %s)", wb.Name, wb.ID)
    }
}
```

### Example: Automatic Token Refresh

```go
package main

import (
    "log"
    "os"
    "time"
    "yourproject/tableau"
)

func main() {
    client := tableau.NewClient(tableau.ClientConfig{
        ServerURL:  os.Getenv("TABLEAU_SERVER"),
        APIVersion: "3.27",
    })

    // Store credentials for automatic refresh
    creds := tableau.AuthCredentials{
        Type:           "pat",
        TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
        TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
        SiteContentUrl: os.Getenv("TABLEAU_SITE"),
    }

    // Initial authentication
    if err := client.SignInWithPAT(creds.TokenName, creds.TokenSecret, creds.SiteContentUrl); err != nil {
        log.Fatal(err)
    }
    defer client.SignOut()

    // Long-running process
    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // Ensure authenticated (will refresh if needed)
            if err := client.EnsureAuthenticated(creds); err != nil {
                log.Printf("Failed to ensure authentication: %v", err)
                continue
            }

            // Make your API calls
            resp, err := client.DoRequest("GET", "/sites/"+client.SiteID, nil)
            if err != nil {
                log.Printf("Request failed: %v", err)
                continue
            }
            resp.Body.Close()

            log.Println("Request successful")
        }
    }
}
```

## Testing

Run the comprehensive test suite:

```bash
go test -v ./tableau/
```

Test coverage includes:
- PAT authentication
- Username/password authentication
- Sign out functionality
- Error handling (401, 403, etc.)
- Token expiry checking
- Automatic token refresh
- Authenticated requests
- Client configuration defaults

## Production Best Practices

### 1. Use Personal Access Tokens

PATs are more secure than username/password, especially in production:

```go
// ✅ GOOD - Use PAT
client.SignInWithPAT(tokenName, tokenSecret, site)

// ❌ AVOID - Username/password in production
client.SignInWithPassword(username, password, site)
```

### 2. Store Credentials Securely

Never hardcode credentials:

```go
// ❌ BAD - Hardcoded credentials
client.SignInWithPAT("my-token", "my-secret", "my-site")

// ✅ GOOD - Environment variables
client.SignInWithPAT(
    os.Getenv("TABLEAU_PAT_NAME"),
    os.Getenv("TABLEAU_PAT_SECRET"),
    os.Getenv("TABLEAU_SITE"),
)

// ✅ EVEN BETTER - Secret management service
// (AWS Secrets Manager, HashiCorp Vault, etc.)
```

### 3. Handle Token Expiry

Implement automatic token refresh for long-running applications:

```go
creds := tableau.AuthCredentials{
    Type:           "pat",
    TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
    TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
    SiteContentUrl: os.Getenv("TABLEAU_SITE"),
}

// Before each request
if err := client.EnsureAuthenticated(creds); err != nil {
    log.Fatal(err)
}
```

### 4. Always Sign Out

Invalidate tokens when done:

```go
client := tableau.NewClient(config)
err := client.SignInWithPAT(tokenName, tokenSecret, site)
if err != nil {
    log.Fatal(err)
}
defer client.SignOut() // Always clean up
```

### 5. Implement Retry Logic

Handle transient failures with exponential backoff:

```go
maxRetries := 3
for attempt := 0; attempt < maxRetries; attempt++ {
    err := client.SignInWithPAT(tokenName, tokenSecret, site)
    if err == nil {
        break
    }

    // Don't retry on invalid credentials
    if tableauErr, ok := err.(*tableau.TableauError); ok {
        if tableauErr.ErrorCode == tableau.ErrorInvalidCredentials {
            return err
        }
    }

    backoff := time.Duration(1<<uint(attempt)) * time.Second
    time.Sleep(backoff)
}
```

### 6. Use Proper Timeouts

Configure appropriate timeouts for your use case:

```go
client := tableau.NewClient(tableau.ClientConfig{
    ServerURL:  os.Getenv("TABLEAU_SERVER"),
    APIVersion: "3.27",
    Timeout:    60 * time.Second, // Adjust based on your needs
})
```

### 7. Log Authentication Events

Monitor authentication for security and debugging:

```go
import "log"

err := client.SignInWithPAT(tokenName, tokenSecret, site)
if err != nil {
    log.Printf("Authentication failed: %v", err)
    return err
}

log.Printf("Successfully authenticated")
log.Printf("Site ID: %s", client.SiteID)
log.Printf("User ID: %s", client.UserID)
log.Printf("Token expires: %v", client.TokenExpiry)
```

## Troubleshooting

### Issue: "401001 Signin Error"

**Cause:** Invalid PAT name or secret

**Solution:**
1. Verify the PAT exists in Tableau Server settings
2. Check the PAT hasn't expired (15 days inactivity or 1 year)
3. Ensure you're copying the complete token secret
4. Verify the token is active (not revoked)

### Issue: "403 Forbidden"

**Cause:** Using token for wrong site

**Solution:**
1. Verify the site content URL matches where you authenticated
2. Check you have permissions for the requested resource
3. Ensure you're not using a token from Site A to access Site B

### Issue: Token Expires During Long Operations

**Cause:** Token lifetime exceeded (240 minutes default)

**Solution:**
```go
// Implement automatic refresh
creds := tableau.AuthCredentials{
    Type:           "pat",
    TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
    TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
    SiteContentUrl: os.Getenv("TABLEAU_SITE"),
}

// Before each operation
if err := client.EnsureAuthenticated(creds); err != nil {
    log.Fatal(err)
}
```

### Issue: SSL Certificate Errors

**Cause:** Self-signed certificates

**Solution (Development Only):**
```go
import (
    "crypto/tls"
    "net/http"
)

// WARNING: Only for development - DO NOT use in production
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true,
    },
}

client := tableau.NewClient(tableau.ClientConfig{
    ServerURL:  "https://tableau.local",
    APIVersion: "3.27",
})
client.HTTPClient.Transport = transport
```

**Production Solution:**
Install the proper CA certificate in your system's trust store.

### Issue: Connection Timeout

**Cause:** Network issues or slow server

**Solution:**
```go
// Increase timeout
client := tableau.NewClient(tableau.ClientConfig{
    ServerURL:  os.Getenv("TABLEAU_SERVER"),
    APIVersion: "3.27",
    Timeout:    120 * time.Second, // Increase timeout
})
```

## Performance Tips

1. **Reuse Client Instances**
   - Create one client and reuse for multiple requests
   - HTTP connection pooling is already configured

2. **Minimize Re-authentication**
   - Cache tokens until near expiry
   - Use `EnsureAuthenticated()` instead of re-authenticating on every request

3. **Concurrent Requests**
   - Make parallel requests for independent operations
   - The client is safe for concurrent use

4. **Request Compression**
   - The client automatically handles gzip compression

5. **Batch Operations**
   - Group multiple operations when possible
   - Use bulk APIs when available in Tableau REST API

## Official Documentation

- **Tableau REST API Reference:** https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref.htm
- **Authentication Concepts:** https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_auth.htm
- **Personal Access Tokens:** https://help.tableau.com/current/server/en-us/security_personal_access_tokens.htm
- **Error Handling:** https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_errors.htm

## API Versions

| API Version | Tableau Version | Status |
|-------------|-----------------|--------|
| 3.27        | 2025.3          | Latest |
| 3.23        | 2024.2          | Stable |
| 3.22        | 2024.2          | Stable |

Always use the latest stable version (3.27) for new implementations.

## License

This implementation is provided as-is for use with the Tableau REST API.

## Contributing

For issues, questions, or contributions, please refer to the official Tableau REST API documentation and community forums.

## Support

- **Tableau Community:** https://community.tableau.com/s/topic/0TO4T000000QF9pWAG/rest-api
- **Official Samples:** https://github.com/tableau/rest-api-samples
- **Developer Portal:** https://www.tableau.com/developer
