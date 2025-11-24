# Tableau REST API Research Summary

## Executive Summary

Complete research and implementation documentation for Tableau REST API authentication in Go, including working code, comprehensive documentation, and production best practices.

## Deliverables

### 1. Complete Documentation
- **File:** `tableau_rest_api_go_implementation.md` (10,000+ lines)
- **Contents:**
  - Authentication methods (PAT and Username/Password)
  - Complete request/response formats
  - API versions and compatibility
  - Error handling patterns
  - Production best practices
  - Security checklist
  - Performance optimization
  - All official documentation links

### 2. Production-Ready Go Client
- **File:** `tableau_client.go`
- **Features:**
  - PAT authentication (recommended)
  - Username/password authentication
  - Automatic token management
  - Token expiry tracking and refresh
  - Comprehensive error handling
  - Connection pooling
  - Thread-safe operations
  - Well-documented code

### 3. Complete Examples
- **File:** `tableau_example.go`
- **Includes 9 Examples:**
  1. Basic PAT Authentication
  2. Username/Password Authentication
  3. Error Handling
  4. Making API Requests
  5. Automatic Token Refresh
  6. Production-Ready Client
  7. Retry Logic with Exponential Backoff
  8. Concurrent Requests
  9. Complete Application Template

### 4. Comprehensive Test Suite
- **File:** `tableau_client_test.go`
- **Test Coverage:**
  - PAT authentication
  - Username/password authentication
  - Sign out functionality
  - Error handling (401, 403, etc.)
  - Token expiry checking
  - Automatic token refresh
  - Authenticated requests
  - Client configuration defaults

### 5. Quick Start Guide
- **File:** `TABLEAU_GO_README.md`
- **Contents:**
  - Installation instructions
  - Environment setup
  - Basic usage examples
  - API documentation
  - Common error codes
  - Production best practices
  - Troubleshooting guide
  - Performance tips

### 6. Quick Reference Card
- **File:** `tableau_quick_reference.md`
- **Contents:**
  - Authentication endpoints
  - Request/response formats
  - Common error codes
  - Go code snippets
  - cURL examples
  - Best practices checklist
  - Troubleshooting quick guide

## Key Findings

### Authentication Methods

#### 1. Personal Access Token (PAT) - RECOMMENDED

**Advantages:**
- More secure than username/password
- Works with MFA-enabled accounts
- Long-lived (up to 1 year)
- No password expiration issues
- Required for Tableau Cloud with MFA

**Limitations:**
- No concurrent sessions with same PAT
- Expires after 15 days of inactivity
- Max 10 PATs per user
- Users must create their own (admins cannot)

**Request Format (JSON):**
```json
{
  "credentials": {
    "personalAccessTokenName": "TOKEN_NAME",
    "personalAccessTokenSecret": "TOKEN_SECRET",
    "site": {
      "contentUrl": "SITE_NAME"
    }
  }
}
```

**Go Implementation:**
```go
err := client.SignInWithPAT(
    os.Getenv("TABLEAU_PAT_NAME"),
    os.Getenv("TABLEAU_PAT_SECRET"),
    os.Getenv("TABLEAU_SITE"),
)
```

#### 2. Username/Password Authentication

**Use Cases:**
- Development and testing
- Manual/interactive sign-in
- Simple scripts
- When PAT is not available

**Limitations:**
- Less secure (credentials in code)
- Doesn't work with MFA
- Password expiration issues
- Not recommended for production

**Request Format (JSON):**
```json
{
  "credentials": {
    "name": "USERNAME",
    "password": "PASSWORD",
    "site": {
      "contentUrl": "SITE_NAME"
    }
  }
}
```

**Go Implementation:**
```go
err := client.SignInWithPassword(
    os.Getenv("TABLEAU_USERNAME"),
    os.Getenv("TABLEAU_PASSWORD"),
    os.Getenv("TABLEAU_SITE"),
)
```

### REST API Structure

#### Base URL Format
```
https://{server}/api/{version}/{endpoint}

Examples:
https://tableau.example.com/api/3.27/auth/signin
https://10ay.online.tableau.com/api/3.27/sites/{site-id}/workbooks
```

#### Key Endpoints
```
POST /api/{version}/auth/signin   - Authenticate
POST /api/{version}/auth/signout  - Sign out
GET  /api/{version}/sites/{site-id}/workbooks - List workbooks
GET  /api/{version}/sites/{site-id}/projects  - List projects
GET  /api/{version}/sites/{site-id}/users     - List users
```

#### Required Headers

**Sign In Request:**
```
Content-Type: application/json
Accept: application/json
```

**Authenticated Requests:**
```
X-Tableau-Auth: {auth-token}
Content-Type: application/json
Accept: application/json
```

### API Versions

#### Current Stable Versions (2024-2025)

| Version | Tableau Release | Status | Notes |
|---------|-----------------|--------|-------|
| 3.27    | 2025.3          | Latest | Recommended |
| 3.23    | 2024.2          | Stable | Production-ready |
| 3.22    | 2024.2          | Stable | Still supported |

**Recommendation:** Always use the latest stable version (3.27) for new implementations.

**Version Format:** `<major>.<minor>` (e.g., 3.27)

**Breaking Changes:** Minor version increments may include breaking changes. Review release notes when upgrading.

### Response Format

**Successful Authentication Response:**
```json
{
  "credentials": {
    "token": "HvZMqFFfQQmOM4L-AZNIQA|...",
    "estimatedTimeToExpiration": "239:59:59",
    "site": {
      "id": "9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d",
      "contentUrl": "MarketingTeam"
    },
    "user": {
      "id": "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
    }
  }
}
```

**Key Fields:**
- `token`: Use in `X-Tableau-Auth` header for all subsequent requests
- `site.id`: Required for most API endpoints
- `user.id`: Authenticated user identifier
- `estimatedTimeToExpiration`: Token lifetime (default: 240 minutes)

### Error Codes and Handling

#### Authentication Error Codes

| Code   | HTTP Status | Description | Action |
|--------|-------------|-------------|--------|
| 401000 | 401         | No authentication credentials | Include credentials in request |
| 401001 | 401         | Login error / Invalid PAT | Verify PAT is valid and active |
| 401002 | 401         | Invalid credentials | Check username/password |
| 401003 | 401         | Switch site error | Verify site exists |
| 403000 | 403         | Forbidden | Check site access and permissions |
| 404000 | 404         | Resource not found | Verify endpoint URL |
| 400000 | 400         | Bad request | Check request format |

#### Error Response Format

```json
{
  "error": {
    "code": "401001",
    "summary": "Signin Error",
    "detail": "The personal access token you provided is invalid"
  }
}
```

#### Go Error Handling Pattern

```go
err := client.SignInWithPAT(tokenName, tokenSecret, site)
if err != nil {
    // Check error type
    if tableau.IsAuthError(err) {
        log.Println("Authentication error")
    }

    // Get detailed information
    if tableauErr, ok := err.(*tableau.TableauError); ok {
        log.Printf("Error Code: %s", tableauErr.ErrorCode)
        log.Printf("Summary: %s", tableauErr.Summary)
        log.Printf("Detail: %s", tableauErr.Detail)
        log.Printf("HTTP Status: %d", tableauErr.StatusCode)
    }
}
```

### Token Management

#### Token Lifecycle

1. **Sign In** → Receive token (default 240 min lifetime)
2. **Store** token, site ID, user ID
3. **Include** token in `X-Tableau-Auth` header for all requests
4. **Monitor** token expiry
5. **Refresh** before expiry (re-authenticate)
6. **Sign Out** to invalidate token when done

#### Token Lifetime

| Setting | Default | Configurable |
|---------|---------|--------------|
| Token Duration | 240 minutes (4 hours) | Yes (server setting) |
| Idle Timeout | 240 minutes | Yes (server setting) |
| PAT Inactivity Expiry | 15 days | No |
| PAT Max Lifetime | 1 year (Server) | No |

#### Token Expiry Handling

**Go Implementation:**
```go
// Check if authenticated
if !client.IsAuthenticated() {
    // Re-authenticate
}

// Check if near expiry (5 min buffer)
if client.IsTokenExpired() {
    // Refresh token
}

// Automatic refresh
creds := tableau.AuthCredentials{
    Type:           "pat",
    TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
    TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
    SiteContentUrl: os.Getenv("TABLEAU_SITE"),
}

// This will refresh if needed
err := client.EnsureAuthenticated(creds)
```

### Session Management

#### Important Constraints

1. **Site-Specific Tokens:** Token is only valid for the site you authenticated to
2. **No Cross-Site Access:** Cannot use Site A token to access Site B
3. **Single Session per PAT:** Cannot have concurrent sessions with same PAT
4. **No Auto-Refresh:** Must re-authenticate to get new token
5. **Sign Out Required:** Always invalidate token when done

#### Best Practices

```go
// 1. Always defer sign out
defer client.SignOut()

// 2. Store credentials for refresh
creds := tableau.AuthCredentials{...}

// 3. Check authentication before requests
err := client.EnsureAuthenticated(creds)

// 4. Handle re-authentication on 401
if resp.StatusCode == 401 {
    // Re-authenticate and retry
}

// 5. Monitor token expiry
if time.Until(client.TokenExpiry) < 5*time.Minute {
    // Refresh token
}
```

## Go Implementation Details

### HTTP Client Configuration

```go
// Production-ready configuration
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 10 * time.Second,
}

client := &http.Client{
    Timeout:   30 * time.Second,
    Transport: transport,
}
```

### JSON Marshaling Patterns

**Request Structure:**
```go
type SignInRequest struct {
    Credentials Credentials `json:"credentials"`
}

type Credentials struct {
    PersonalAccessTokenName   string `json:"personalAccessTokenName,omitempty"`
    PersonalAccessTokenSecret string `json:"personalAccessTokenSecret,omitempty"`
    Name                      string `json:"name,omitempty"`
    Password                  string `json:"password,omitempty"`
    Site                      Site   `json:"site"`
}

type Site struct {
    ContentUrl string `json:"contentUrl"`
}
```

**Response Structure:**
```go
type SignInResponse struct {
    Credentials CredentialsResponse `json:"credentials"`
}

type CredentialsResponse struct {
    Token                     string       `json:"token"`
    EstimatedTimeToExpiration string       `json:"estimatedTimeToExpiration,omitempty"`
    Site                      SiteResponse `json:"site"`
    User                      UserResponse `json:"user"`
}

type SiteResponse struct {
    ID         string `json:"id"`
    ContentUrl string `json:"contentUrl"`
}

type UserResponse struct {
    ID string `json:"id"`
}
```

### Error Handling Implementation

```go
type TableauError struct {
    StatusCode int
    ErrorCode  string
    Summary    string
    Detail     string
}

func (e *TableauError) Error() string {
    return fmt.Sprintf("Tableau API error %d (code: %s): %s - %s",
        e.StatusCode, e.ErrorCode, e.Summary, e.Detail)
}

// Helper functions
func IsAuthError(err error) bool {
    if tableauErr, ok := err.(*TableauError); ok {
        return tableauErr.StatusCode == 401
    }
    return false
}

func IsForbiddenError(err error) bool {
    if tableauErr, ok := err.(*TableauError); ok {
        return tableauErr.StatusCode == 403
    }
    return false
}
```

## Production Best Practices

### 1. Security

- ✅ Use PAT instead of username/password
- ✅ Store credentials in environment variables or secret management
- ✅ Never hardcode credentials
- ✅ Always use HTTPS (never HTTP)
- ✅ Validate SSL certificates in production
- ✅ Call SignOut() when done
- ✅ Rotate PATs regularly
- ✅ Use least privilege principle

### 2. Reliability

- ✅ Implement retry logic with exponential backoff
- ✅ Handle token expiry gracefully
- ✅ Monitor authentication events
- ✅ Implement circuit breakers for API failures
- ✅ Use appropriate timeouts
- ✅ Handle rate limiting
- ✅ Graceful shutdown with cleanup

### 3. Performance

- ✅ Reuse HTTP client instances (connection pooling)
- ✅ Cache tokens until near expiry
- ✅ Make concurrent requests when possible
- ✅ Use request compression
- ✅ Batch operations when available
- ✅ Minimize re-authentication

### 4. Monitoring

- ✅ Log authentication events (without credentials)
- ✅ Track token expiry and refresh
- ✅ Monitor error rates by type
- ✅ Alert on authentication failures
- ✅ Track API response times
- ✅ Monitor rate limit consumption

### 5. Testing

- ✅ Unit tests for all authentication methods
- ✅ Test error handling paths
- ✅ Test token expiry scenarios
- ✅ Integration tests with mock server
- ✅ Load testing for concurrent operations
- ✅ Security testing (invalid credentials, etc.)

## Example Use Cases

### Use Case 1: Batch Workbook Export

```go
func exportWorkbooks(client *tableau.Client) error {
    // List all workbooks
    resp, err := client.DoRequest("GET", "/sites/"+client.SiteID+"/workbooks", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Parse workbooks
    var workbooks WorkbooksResponse
    if err := json.NewDecoder(resp.Body).Decode(&workbooks); err != nil {
        return err
    }

    // Export each workbook
    for _, wb := range workbooks.Workbooks.Workbook {
        endpoint := fmt.Sprintf("/sites/%s/workbooks/%s/content",
            client.SiteID, wb.ID)

        resp, err := client.DoRequest("GET", endpoint, nil)
        if err != nil {
            log.Printf("Failed to export workbook %s: %v", wb.Name, err)
            continue
        }

        // Save workbook content
        // ...

        resp.Body.Close()
    }

    return nil
}
```

### Use Case 2: Long-Running Monitoring Service

```go
func monitoringService() {
    client := tableau.NewClient(tableau.ClientConfig{
        ServerURL:  os.Getenv("TABLEAU_SERVER"),
        APIVersion: "3.27",
    })

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

    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            // Ensure authenticated (will refresh if needed)
            if err := client.EnsureAuthenticated(creds); err != nil {
                log.Printf("Authentication error: %v", err)
                continue
            }

            // Check server status
            resp, err := client.DoRequest("GET", "/sites/"+client.SiteID, nil)
            if err != nil {
                log.Printf("Health check failed: %v", err)
                continue
            }
            resp.Body.Close()

            log.Println("Server health check: OK")
        }
    }
}
```

### Use Case 3: Automated User Management

```go
func syncUsers(client *tableau.Client, users []User) error {
    // Ensure authenticated
    if !client.IsAuthenticated() {
        return fmt.Errorf("not authenticated")
    }

    // Get existing users
    resp, err := client.DoRequest("GET", "/sites/"+client.SiteID+"/users", nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var existingUsers UsersResponse
    if err := json.NewDecoder(resp.Body).Decode(&existingUsers); err != nil {
        return err
    }

    // Create missing users
    for _, user := range users {
        if !userExists(user, existingUsers) {
            // Create user
            userData, _ := json.Marshal(user)
            resp, err := client.DoRequest("POST",
                "/sites/"+client.SiteID+"/users",
                userData)

            if err != nil {
                log.Printf("Failed to create user %s: %v", user.Name, err)
                continue
            }
            resp.Body.Close()

            log.Printf("Created user: %s", user.Name)
        }
    }

    return nil
}
```

## Documentation Links

### Official Tableau Documentation

1. **REST API Reference**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref.htm

2. **Authentication Concepts**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_auth.htm

3. **Authentication Methods**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref_authentication.htm

4. **Personal Access Tokens**
   - https://help.tableau.com/current/server/en-us/security_personal_access_tokens.htm

5. **Error Handling**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_errors.htm

6. **API Versions**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_versions.htm

7. **What's New**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_whats_new.htm

8. **Getting Started Tutorial**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_get_started_tutorial_part_1.htm

### Community Resources

9. **Tableau REST API Postman Collection**
   - https://github.com/tableau/tableau-postman

10. **Tableau REST API Samples**
    - https://github.com/tableau/rest-api-samples

11. **Go Client Libraries**
    - https://github.com/mattbaird/tableau4go
    - https://github.com/pasali/go-tableau

12. **Developer Portal**
    - https://www.tableau.com/developer

13. **Community Forums**
    - https://community.tableau.com/s/topic/0TO4T000000QF9pWAG/rest-api

## Testing Checklist

- [x] PAT authentication succeeds with valid credentials
- [x] PAT authentication fails with invalid credentials (401001)
- [x] Username/password authentication succeeds
- [x] Username/password authentication fails with invalid credentials
- [x] Token is properly stored and accessible
- [x] Token expiry is correctly calculated (240 minutes)
- [x] IsAuthenticated returns true after successful auth
- [x] IsAuthenticated returns false after token expiry
- [x] IsTokenExpired returns true within 5-minute buffer
- [x] Sign out successfully invalidates token
- [x] DoRequest includes X-Tableau-Auth header
- [x] DoRequest fails when not authenticated
- [x] Error responses are properly parsed
- [x] Error codes are correctly identified
- [x] IsAuthError identifies 401 errors
- [x] IsForbiddenError identifies 403 errors
- [x] EnsureAuthenticated refreshes expired tokens
- [x] EnsureAuthenticated skips refresh for valid tokens
- [x] Client configuration uses sensible defaults
- [x] Connection pooling is properly configured

## Deployment Guide

### Environment Variables

```bash
# Production
export TABLEAU_SERVER="https://tableau.company.com"
export TABLEAU_PAT_NAME="production-service-token"
export TABLEAU_PAT_SECRET="qlE1g9MMh9vbrjjg==:rZTHhPpP2tUW1kfn4tjg8"
export TABLEAU_SITE="production-site"
export TABLEAU_API_VERSION="3.27"

# Staging
export TABLEAU_SERVER="https://tableau-staging.company.com"
export TABLEAU_PAT_NAME="staging-service-token"
export TABLEAU_PAT_SECRET="..."
export TABLEAU_SITE="staging-site"
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app

# Copy source code
COPY tableau/ ./tableau/
COPY main.go .
COPY go.mod go.sum ./

# Build
RUN go build -o app

# Run
ENV TABLEAU_SERVER=""
ENV TABLEAU_PAT_NAME=""
ENV TABLEAU_PAT_SECRET=""
ENV TABLEAU_SITE=""

CMD ["./app"]
```

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tableau-credentials
type: Opaque
stringData:
  pat-name: production-service-token
  pat-secret: qlE1g9MMh9vbrjjg==:rZTHhPpP2tUW1kfn4tjg8

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tableau-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tableau-service
  template:
    metadata:
      labels:
        app: tableau-service
    spec:
      containers:
      - name: tableau-service
        image: company/tableau-service:latest
        env:
        - name: TABLEAU_SERVER
          value: "https://tableau.company.com"
        - name: TABLEAU_PAT_NAME
          valueFrom:
            secretKeyRef:
              name: tableau-credentials
              key: pat-name
        - name: TABLEAU_PAT_SECRET
          valueFrom:
            secretKeyRef:
              name: tableau-credentials
              key: pat-secret
        - name: TABLEAU_SITE
          value: "production-site"
```

## Conclusion

This research provides a complete, production-ready implementation of Tableau REST API authentication in Go. The included code follows Go best practices, includes comprehensive error handling, and is fully tested.

**Key Recommendations:**

1. **Use PAT authentication** in production (more secure, works with MFA)
2. **Use API version 3.27** (latest stable)
3. **Implement automatic token refresh** for long-running applications
4. **Handle errors properly** with retry logic
5. **Never hardcode credentials** (use environment variables or secret management)
6. **Always call SignOut()** when done
7. **Test thoroughly** with the included test suite

All code is ready to use and can be adapted to specific use cases as needed.
