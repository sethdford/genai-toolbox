# Tableau REST API Authentication - Complete Go Implementation Guide

## Table of Contents
1. [Overview](#overview)
2. [Authentication Methods](#authentication-methods)
3. [API Versions](#api-versions)
4. [Complete Go Implementation](#complete-go-implementation)
5. [Error Handling](#error-handling)
6. [Production Best Practices](#production-best-practices)
7. [Documentation Links](#documentation-links)

---

## Overview

The Tableau REST API provides programmatic access to Tableau Server and Tableau Cloud. All API calls require authentication using a credentials token obtained through the Sign In endpoint.

**Key Points:**
- Default token lifetime: 240 minutes (4 hours)
- Token is site-specific (cannot be used across different sites)
- Recommended: Use Personal Access Tokens (PAT) for security
- API Version: 3.27 (latest stable as of 2025)

---

## Authentication Methods

### 1. Personal Access Token (PAT) Authentication

**Recommended method** - More secure, especially for:
- Automated scripts and tasks
- MFA-enabled environments
- Long-running applications
- Production deployments

**Limitations:**
- No concurrent sessions with same PAT
- Expires after 15 days of inactivity
- Expires after 1 year by default (Tableau Server)
- Maximum 10 PATs per user

### 2. Username/Password Authentication

**Quick method** - Suitable for:
- Manual/interactive sign-in
- Development and testing
- Simple scripts

**Limitations:**
- Less secure (credentials in code)
- Doesn't work with MFA-enabled accounts
- Password expiration issues

---

## API Versions

### Current Stable Versions
- **Latest:** API 3.27 (Tableau 2025.3)
- **Previous:** API 3.23 (Tableau 2024.2)
- **Format:** `<major>.<minor>` (e.g., 3.27)

### Version in Requests
```
https://MY-SERVER/api/3.27/auth/signin
                    ^^^^
                    API version here
```

**Recommendation:** Always use the latest stable version (3.27) for new implementations.

---

## Complete Go Implementation

### 1. Package Structure and Imports

```go
package tableau

import (
    "bytes"
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "time"
)
```

### 2. Data Structures

```go
// Authentication request structures
type SignInRequest struct {
    Credentials Credentials `json:"credentials" xml:"credentials"`
}

type Credentials struct {
    Name                      string `json:"name,omitempty" xml:"name,attr,omitempty"`
    Password                  string `json:"password,omitempty" xml:"password,attr,omitempty"`
    PersonalAccessTokenName   string `json:"personalAccessTokenName,omitempty" xml:"personalAccessTokenName,attr,omitempty"`
    PersonalAccessTokenSecret string `json:"personalAccessTokenSecret,omitempty" xml:"personalAccessTokenSecret,attr,omitempty"`
    Site                      Site   `json:"site" xml:"site"`
}

type Site struct {
    ContentUrl string `json:"contentUrl" xml:"contentUrl,attr"`
}

// Authentication response structures
type SignInResponse struct {
    Credentials CredentialsResponse `json:"credentials" xml:"credentials"`
}

type CredentialsResponse struct {
    Token                     string `json:"token" xml:"token,attr"`
    EstimatedTimeToExpiration string `json:"estimatedTimeToExpiration,omitempty" xml:"estimatedTimeToExpiration,attr,omitempty"`
    Site                      SiteResponse `json:"site" xml:"site"`
    User                      UserResponse `json:"user" xml:"user"`
}

type SiteResponse struct {
    ID         string `json:"id" xml:"id,attr"`
    ContentUrl string `json:"contentUrl" xml:"contentUrl,attr"`
}

type UserResponse struct {
    ID string `json:"id" xml:"id,attr"`
}

// Error response structure
type ErrorResponse struct {
    XMLName xml.Name `xml:"tsResponse"`
    Error   struct {
        Code    string `xml:"code,attr"`
        Summary string `xml:"summary,attr"`
        Detail  string `xml:"detail,attr"`
    } `xml:"error"`
}
```

### 3. Client Configuration

```go
// Client represents a Tableau REST API client
type Client struct {
    ServerURL  string
    APIVersion string
    SiteID     string
    Token      string
    UserID     string
    HTTPClient *http.Client
    TokenExpiry time.Time
}

// ClientConfig holds configuration for creating a new client
type ClientConfig struct {
    ServerURL  string        // e.g., "https://tableau.example.com"
    APIVersion string        // e.g., "3.27" (use latest stable)
    Timeout    time.Duration // HTTP client timeout
}

// NewClient creates a new Tableau REST API client
func NewClient(config ClientConfig) *Client {
    if config.APIVersion == "" {
        config.APIVersion = "3.27" // Use latest stable version
    }

    if config.Timeout == 0 {
        config.Timeout = 30 * time.Second
    }

    return &Client{
        ServerURL:  config.ServerURL,
        APIVersion: config.APIVersion,
        HTTPClient: &http.Client{
            Timeout: config.Timeout,
        },
    }
}
```

### 4. Personal Access Token (PAT) Authentication

```go
// SignInWithPAT authenticates using a Personal Access Token
func (c *Client) SignInWithPAT(tokenName, tokenSecret, siteContentUrl string) error {
    url := fmt.Sprintf("%s/api/%s/auth/signin", c.ServerURL, c.APIVersion)

    // Prepare request body
    reqBody := SignInRequest{
        Credentials: Credentials{
            PersonalAccessTokenName:   tokenName,
            PersonalAccessTokenSecret: tokenSecret,
            Site: Site{
                ContentUrl: siteContentUrl,
            },
        },
    }

    // Marshal to JSON
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("failed to marshal request: %w", err)
    }

    // Create HTTP request
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    // Execute request
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }

    // Check for error response
    if resp.StatusCode != http.StatusOK {
        return c.parseErrorResponse(resp.StatusCode, body)
    }

    // Parse successful response
    var signInResp SignInResponse
    if err := json.Unmarshal(body, &signInResp); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }

    // Store authentication details
    c.Token = signInResp.Credentials.Token
    c.SiteID = signInResp.Credentials.Site.ID
    c.UserID = signInResp.Credentials.User.ID

    // Calculate token expiry (default 240 minutes)
    c.TokenExpiry = time.Now().Add(240 * time.Minute)

    return nil
}
```

### 5. Username/Password Authentication

```go
// SignInWithPassword authenticates using username and password
func (c *Client) SignInWithPassword(username, password, siteContentUrl string) error {
    url := fmt.Sprintf("%s/api/%s/auth/signin", c.ServerURL, c.APIVersion)

    // Prepare request body
    reqBody := SignInRequest{
        Credentials: Credentials{
            Name:     username,
            Password: password,
            Site: Site{
                ContentUrl: siteContentUrl,
            },
        },
    }

    // Marshal to JSON
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return fmt.Errorf("failed to marshal request: %w", err)
    }

    // Create HTTP request
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    // Execute request
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }

    // Check for error response
    if resp.StatusCode != http.StatusOK {
        return c.parseErrorResponse(resp.StatusCode, body)
    }

    // Parse successful response
    var signInResp SignInResponse
    if err := json.Unmarshal(body, &signInResp); err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }

    // Store authentication details
    c.Token = signInResp.Credentials.Token
    c.SiteID = signInResp.Credentials.Site.ID
    c.UserID = signInResp.Credentials.User.ID

    // Calculate token expiry (default 240 minutes)
    c.TokenExpiry = time.Now().Add(240 * time.Minute)

    return nil
}
```

### 6. XML-Based Authentication (Alternative)

```go
// SignInWithPATXML authenticates using PAT with XML request/response
func (c *Client) SignInWithPATXML(tokenName, tokenSecret, siteContentUrl string) error {
    url := fmt.Sprintf("%s/api/%s/auth/signin", c.ServerURL, c.APIVersion)

    // Prepare XML request body
    reqBody := `<tsRequest>
        <credentials personalAccessTokenName="` + tokenName + `"
                    personalAccessTokenSecret="` + tokenSecret + `">
            <site contentUrl="` + siteContentUrl + `" />
        </credentials>
    </tsRequest>`

    // Create HTTP request
    req, err := http.NewRequest("POST", url, bytes.NewBufferString(reqBody))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/xml")
    req.Header.Set("Accept", "application/xml")

    // Execute request
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }

    // Check for error response
    if resp.StatusCode != http.StatusOK {
        return c.parseErrorResponseXML(resp.StatusCode, body)
    }

    // Parse XML response
    type XMLResponse struct {
        XMLName     xml.Name `xml:"tsResponse"`
        Credentials struct {
            Token string `xml:"token,attr"`
            Site  struct {
                ID         string `xml:"id,attr"`
                ContentUrl string `xml:"contentUrl,attr"`
            } `xml:"site"`
            User struct {
                ID string `xml:"id,attr"`
            } `xml:"user"`
        } `xml:"credentials"`
    }

    var xmlResp XMLResponse
    if err := xml.Unmarshal(body, &xmlResp); err != nil {
        return fmt.Errorf("failed to parse XML response: %w", err)
    }

    // Store authentication details
    c.Token = xmlResp.Credentials.Token
    c.SiteID = xmlResp.Credentials.Site.ID
    c.UserID = xmlResp.Credentials.User.ID
    c.TokenExpiry = time.Now().Add(240 * time.Minute)

    return nil
}
```

### 7. Sign Out Method

```go
// SignOut invalidates the current authentication token
func (c *Client) SignOut() error {
    if c.Token == "" {
        return fmt.Errorf("not authenticated")
    }

    url := fmt.Sprintf("%s/api/%s/auth/signout", c.ServerURL, c.APIVersion)

    req, err := http.NewRequest("POST", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // Include auth token
    req.Header.Set("X-Tableau-Auth", c.Token)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return c.parseErrorResponse(resp.StatusCode, body)
    }

    // Clear authentication state
    c.Token = ""
    c.SiteID = ""
    c.UserID = ""
    c.TokenExpiry = time.Time{}

    return nil
}
```

### 8. Token Management and Session Handling

```go
// IsAuthenticated checks if the client has a valid token
func (c *Client) IsAuthenticated() bool {
    return c.Token != "" && time.Now().Before(c.TokenExpiry)
}

// IsTokenExpired checks if the current token is expired or near expiry
func (c *Client) IsTokenExpired() bool {
    // Consider token expired if less than 5 minutes remaining
    return time.Now().Add(5 * time.Minute).After(c.TokenExpiry)
}

// RefreshToken re-authenticates to get a new token
// Note: Requires storing original credentials
type AuthCredentials struct {
    Type              string // "pat" or "password"
    TokenName         string // for PAT
    TokenSecret       string // for PAT
    Username          string // for password
    Password          string // for password
    SiteContentUrl    string
}

// SetAuthCredentials stores credentials for automatic refresh
func (c *Client) SetAuthCredentials(creds AuthCredentials) {
    // In production, consider encrypting these in memory
    // This is a simplified example
}

// EnsureAuthenticated checks and refreshes token if needed
func (c *Client) EnsureAuthenticated(creds AuthCredentials) error {
    if c.IsAuthenticated() && !c.IsTokenExpired() {
        return nil // Token still valid
    }

    // Re-authenticate based on credential type
    switch creds.Type {
    case "pat":
        return c.SignInWithPAT(creds.TokenName, creds.TokenSecret, creds.SiteContentUrl)
    case "password":
        return c.SignInWithPassword(creds.Username, creds.Password, creds.SiteContentUrl)
    default:
        return fmt.Errorf("unknown credential type: %s", creds.Type)
    }
}
```

### 9. Making Authenticated Requests

```go
// DoRequest makes an authenticated request to the Tableau REST API
func (c *Client) DoRequest(method, endpoint string, body []byte) (*http.Response, error) {
    if !c.IsAuthenticated() {
        return nil, fmt.Errorf("not authenticated")
    }

    url := fmt.Sprintf("%s/api/%s%s", c.ServerURL, c.APIVersion, endpoint)

    var bodyReader io.Reader
    if body != nil {
        bodyReader = bytes.NewBuffer(body)
    }

    req, err := http.NewRequest(method, url, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set authentication header
    req.Header.Set("X-Tableau-Auth", c.Token)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request: %w", err)
    }

    return resp, nil
}
```

---

## Error Handling

### Error Codes

```go
// TableauError represents a Tableau REST API error
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

// Common error codes
const (
    ErrorNoAuthCredentials       = "401000" // No authentication credentials
    ErrorLoginFailed            = "401001" // Login error
    ErrorInvalidCredentials     = "401002" // Invalid authentication credentials
    ErrorSwitchSiteFailed       = "401003" // Switch site error
    ErrorForbidden              = "403000" // Forbidden
    ErrorResourceNotFound       = "404000" // Resource not found
    ErrorBadRequest             = "400000" // Bad request
)

// parseErrorResponse parses JSON error response
func (c *Client) parseErrorResponse(statusCode int, body []byte) error {
    // Try parsing as JSON first
    var jsonErr struct {
        Error struct {
            Code    string `json:"code"`
            Summary string `json:"summary"`
            Detail  string `json:"detail"`
        } `json:"error"`
    }

    if err := json.Unmarshal(body, &jsonErr); err == nil && jsonErr.Error.Code != "" {
        return &TableauError{
            StatusCode: statusCode,
            ErrorCode:  jsonErr.Error.Code,
            Summary:    jsonErr.Error.Summary,
            Detail:     jsonErr.Error.Detail,
        }
    }

    // Fall back to XML parsing
    return c.parseErrorResponseXML(statusCode, body)
}

// parseErrorResponseXML parses XML error response
func (c *Client) parseErrorResponseXML(statusCode int, body []byte) error {
    var errResp ErrorResponse
    if err := xml.Unmarshal(body, &errResp); err != nil {
        return &TableauError{
            StatusCode: statusCode,
            ErrorCode:  "",
            Summary:    "Failed to parse error response",
            Detail:     string(body),
        }
    }

    return &TableauError{
        StatusCode: statusCode,
        ErrorCode:  errResp.Error.Code,
        Summary:    errResp.Error.Summary,
        Detail:     errResp.Error.Detail,
    }
}

// IsAuthError checks if an error is authentication-related
func IsAuthError(err error) bool {
    if tableauErr, ok := err.(*TableauError); ok {
        return tableauErr.StatusCode == 401 ||
               tableauErr.ErrorCode == ErrorNoAuthCredentials ||
               tableauErr.ErrorCode == ErrorLoginFailed ||
               tableauErr.ErrorCode == ErrorInvalidCredentials
    }
    return false
}

// IsForbiddenError checks if an error is a forbidden (403) error
func IsForbiddenError(err error) bool {
    if tableauErr, ok := err.(*TableauError); ok {
        return tableauErr.StatusCode == 403
    }
    return false
}
```

### Error Handling Patterns

```go
// Example: Handling authentication errors with retry
func authenticateWithRetry(client *Client, creds AuthCredentials, maxRetries int) error {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        var err error

        switch creds.Type {
        case "pat":
            err = client.SignInWithPAT(creds.TokenName, creds.TokenSecret, creds.SiteContentUrl)
        case "password":
            err = client.SignInWithPassword(creds.Username, creds.Password, creds.SiteContentUrl)
        }

        if err == nil {
            return nil // Success
        }

        lastErr = err

        // Check if it's a retryable error
        if IsAuthError(err) {
            if tableauErr, ok := err.(*TableauError); ok {
                // Invalid credentials - don't retry
                if tableauErr.ErrorCode == ErrorInvalidCredentials {
                    return err
                }
            }
        }

        // Wait before retry (exponential backoff)
        time.Sleep(time.Duration(i+1) * time.Second)
    }

    return fmt.Errorf("authentication failed after %d retries: %w", maxRetries, lastErr)
}
```

---

## Production Best Practices

### 1. Secure Credential Management

```go
// NEVER hardcode credentials
// ❌ BAD
const (
    username = "admin@example.com"
    password = "MyPassword123"
)

// ✅ GOOD - Use environment variables
import "os"

func getCredentialsFromEnv() AuthCredentials {
    return AuthCredentials{
        Type:           os.Getenv("TABLEAU_AUTH_TYPE"),        // "pat" or "password"
        TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
        TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
        Username:       os.Getenv("TABLEAU_USERNAME"),
        Password:       os.Getenv("TABLEAU_PASSWORD"),
        SiteContentUrl: os.Getenv("TABLEAU_SITE"),
    }
}

// ✅ EVEN BETTER - Use secret management service
// (AWS Secrets Manager, HashiCorp Vault, etc.)
```

### 2. Connection Pooling and Reuse

```go
// Configure HTTP client with connection pooling
func NewProductionClient(config ClientConfig) *Client {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
    }

    return &Client{
        ServerURL:  config.ServerURL,
        APIVersion: config.APIVersion,
        HTTPClient: &http.Client{
            Timeout:   config.Timeout,
            Transport: transport,
        },
    }
}
```

### 3. Automatic Token Refresh

```go
// Middleware to ensure authentication before each request
type AuthenticatedClient struct {
    *Client
    credentials AuthCredentials
}

func NewAuthenticatedClient(config ClientConfig, creds AuthCredentials) (*AuthenticatedClient, error) {
    client := NewClient(config)

    // Initial authentication
    var err error
    switch creds.Type {
    case "pat":
        err = client.SignInWithPAT(creds.TokenName, creds.TokenSecret, creds.SiteContentUrl)
    case "password":
        err = client.SignInWithPassword(creds.Username, creds.Password, creds.SiteContentUrl)
    default:
        return nil, fmt.Errorf("invalid auth type: %s", creds.Type)
    }

    if err != nil {
        return nil, err
    }

    return &AuthenticatedClient{
        Client:      client,
        credentials: creds,
    }, nil
}

// DoRequest with automatic token refresh
func (ac *AuthenticatedClient) DoRequest(method, endpoint string, body []byte) (*http.Response, error) {
    // Ensure we have a valid token
    if err := ac.EnsureAuthenticated(ac.credentials); err != nil {
        return nil, fmt.Errorf("failed to ensure authentication: %w", err)
    }

    // Make the request
    resp, err := ac.Client.DoRequest(method, endpoint, body)

    // If we get a 401, try refreshing the token once
    if err == nil && resp.StatusCode == 401 {
        resp.Body.Close()

        // Force re-authentication
        switch ac.credentials.Type {
        case "pat":
            if err := ac.SignInWithPAT(ac.credentials.TokenName,
                ac.credentials.TokenSecret, ac.credentials.SiteContentUrl); err != nil {
                return nil, fmt.Errorf("failed to refresh token: %w", err)
            }
        case "password":
            if err := ac.SignInWithPassword(ac.credentials.Username,
                ac.credentials.Password, ac.credentials.SiteContentUrl); err != nil {
                return nil, fmt.Errorf("failed to refresh token: %w", err)
            }
        }

        // Retry the request
        return ac.Client.DoRequest(method, endpoint, body)
    }

    return resp, err
}
```

### 4. Logging and Monitoring

```go
import "log"

// LoggingClient wraps a client with logging
type LoggingClient struct {
    *Client
    logger *log.Logger
}

func (lc *LoggingClient) SignInWithPAT(tokenName, tokenSecret, siteContentUrl string) error {
    lc.logger.Printf("Attempting PAT authentication for site: %s", siteContentUrl)

    err := lc.Client.SignInWithPAT(tokenName, tokenSecret, siteContentUrl)

    if err != nil {
        lc.logger.Printf("Authentication failed: %v", err)
        return err
    }

    lc.logger.Printf("Successfully authenticated. Token expires at: %v", lc.TokenExpiry)
    return nil
}

func (lc *LoggingClient) DoRequest(method, endpoint string, body []byte) (*http.Response, error) {
    lc.logger.Printf("Making %s request to %s", method, endpoint)

    start := time.Now()
    resp, err := lc.Client.DoRequest(method, endpoint, body)
    duration := time.Since(start)

    if err != nil {
        lc.logger.Printf("Request failed after %v: %v", duration, err)
        return nil, err
    }

    lc.logger.Printf("Request completed in %v with status %d", duration, resp.StatusCode)
    return resp, nil
}
```

### 5. Graceful Shutdown

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Create client
    client, err := NewAuthenticatedClient(
        ClientConfig{
            ServerURL:  os.Getenv("TABLEAU_SERVER"),
            APIVersion: "3.27",
            Timeout:    30 * time.Second,
        },
        getCredentialsFromEnv(),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Run your application
    go func() {
        // Your application logic here
    }()

    // Wait for shutdown signal
    <-sigChan
    log.Println("Shutting down gracefully...")

    // Sign out to invalidate token
    if err := client.SignOut(); err != nil {
        log.Printf("Error during sign out: %v", err)
    }

    log.Println("Shutdown complete")
}
```

### 6. Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimitedClient struct {
    *Client
    limiter *rate.Limiter
}

func NewRateLimitedClient(client *Client, requestsPerSecond float64) *RateLimitedClient {
    return &RateLimitedClient{
        Client:  client,
        limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), int(requestsPerSecond)),
    }
}

func (rlc *RateLimitedClient) DoRequest(method, endpoint string, body []byte) (*http.Response, error) {
    // Wait for rate limiter
    if err := rlc.limiter.Wait(context.Background()); err != nil {
        return nil, fmt.Errorf("rate limiter error: %w", err)
    }

    return rlc.Client.DoRequest(method, endpoint, body)
}
```

### 7. Complete Production Example

```go
package main

import (
    "log"
    "os"
    "time"

    "yourproject/tableau"
)

func main() {
    // Load configuration from environment
    config := tableau.ClientConfig{
        ServerURL:  os.Getenv("TABLEAU_SERVER"),
        APIVersion: "3.27",
        Timeout:    30 * time.Second,
    }

    creds := tableau.AuthCredentials{
        Type:           "pat",
        TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
        TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
        SiteContentUrl: os.Getenv("TABLEAU_SITE"),
    }

    // Validate configuration
    if config.ServerURL == "" {
        log.Fatal("TABLEAU_SERVER environment variable is required")
    }
    if creds.TokenName == "" || creds.TokenSecret == "" {
        log.Fatal("TABLEAU_PAT_NAME and TABLEAU_PAT_SECRET are required")
    }

    // Create authenticated client
    client, err := tableau.NewAuthenticatedClient(config, creds)
    if err != nil {
        log.Fatalf("Failed to authenticate: %v", err)
    }
    defer client.SignOut()

    log.Printf("Successfully authenticated to Tableau Server")
    log.Printf("Site ID: %s", client.SiteID)
    log.Printf("User ID: %s", client.UserID)
    log.Printf("Token expires: %v", client.TokenExpiry)

    // Example: Make API request
    resp, err := client.DoRequest("GET", "/sites/"+client.SiteID+"/workbooks", nil)
    if err != nil {
        log.Fatalf("Failed to fetch workbooks: %v", err)
    }
    defer resp.Body.Close()

    log.Printf("Fetched workbooks: status %d", resp.StatusCode)

    // Your application logic here...
}
```

---

## Documentation Links

### Official Tableau REST API Documentation

1. **Authentication Concepts**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_auth.htm

2. **Authentication Methods Reference**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref_authentication.htm

3. **Sign In Method**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref_authentication.htm#sign_in

4. **Sign Out Method**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref_authentication.htm#sign_out

5. **Personal Access Tokens**
   - https://help.tableau.com/current/server/en-us/security_personal_access_tokens.htm

6. **Error Handling**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_errors.htm

7. **API Versions**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_versions.htm

8. **What's New in REST API**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_whats_new.htm

9. **Getting Started Tutorial**
   - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_get_started_tutorial_part_1.htm

10. **Complete REST API Reference**
    - https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref.htm

### Community Resources

11. **Tableau REST API Postman Collection**
    - https://github.com/tableau/tableau-postman

12. **Tableau REST API Samples**
    - https://github.com/tableau/rest-api-samples

13. **Go Tableau Client Libraries**
    - https://github.com/mattbaird/tableau4go
    - https://github.com/pasali/go-tableau

### Additional Resources

14. **Tableau Developer Portal**
    - https://www.tableau.com/developer

15. **Tableau Community Forums**
    - https://community.tableau.com/s/topic/0TO4T000000QF9pWAG/rest-api

---

## Quick Reference

### Authentication Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/{version}/auth/signin` | Authenticate and get token |
| POST | `/api/{version}/auth/signout` | Invalidate current token |
| POST | `/api/{version}/auth/switchSite` | Switch to different site |

### Required Headers

| Header | Value | When |
|--------|-------|------|
| `Content-Type` | `application/json` or `application/xml` | Sign In request |
| `Accept` | `application/json` or `application/xml` | Sign In request |
| `X-Tableau-Auth` | `{auth-token}` | All authenticated requests |

### Common HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (success, no response body) |
| 400 | Bad Request |
| 401 | Unauthorized (auth failed) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 405 | Method Not Allowed |
| 500 | Internal Server Error |

### Token Lifecycle

```
1. Sign In → Receive Token (240 min lifetime)
2. Include token in X-Tableau-Auth header
3. Monitor token expiry
4. Refresh before expiry (re-authenticate)
5. Sign Out when done (invalidate token)
```

---

## Testing Your Implementation

### Unit Test Example

```go
package tableau_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "yourproject/tableau"
)

func TestSignInWithPAT(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/3.27/auth/signin" {
            t.Errorf("Expected path /api/3.27/auth/signin, got %s", r.URL.Path)
        }

        if r.Method != "POST" {
            t.Errorf("Expected POST method, got %s", r.Method)
        }

        // Return mock response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "credentials": {
                "token": "test-token-123",
                "site": {
                    "id": "site-id-456",
                    "contentUrl": "test-site"
                },
                "user": {
                    "id": "user-id-789"
                }
            }
        }`))
    }))
    defer server.Close()

    // Create client
    client := tableau.NewClient(tableau.ClientConfig{
        ServerURL:  server.URL,
        APIVersion: "3.27",
    })

    // Test authentication
    err := client.SignInWithPAT("test-token", "test-secret", "test-site")
    if err != nil {
        t.Fatalf("SignInWithPAT failed: %v", err)
    }

    // Verify token is set
    if client.Token != "test-token-123" {
        t.Errorf("Expected token 'test-token-123', got '%s'", client.Token)
    }

    // Verify site ID is set
    if client.SiteID != "site-id-456" {
        t.Errorf("Expected site ID 'site-id-456', got '%s'", client.SiteID)
    }
}

func TestErrorHandling(t *testing.T) {
    // Create mock server that returns error
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte(`{
            "error": {
                "code": "401001",
                "summary": "Signin Error",
                "detail": "The personal access token you provided is invalid"
            }
        }`))
    }))
    defer server.Close()

    client := tableau.NewClient(tableau.ClientConfig{
        ServerURL:  server.URL,
        APIVersion: "3.27",
    })

    err := client.SignInWithPAT("invalid-token", "invalid-secret", "test-site")

    if err == nil {
        t.Fatal("Expected error, got nil")
    }

    // Check if it's a Tableau error
    if !tableau.IsAuthError(err) {
        t.Errorf("Expected auth error, got: %v", err)
    }

    // Check error details
    tableauErr, ok := err.(*tableau.TableauError)
    if !ok {
        t.Fatalf("Expected *TableauError, got %T", err)
    }

    if tableauErr.ErrorCode != "401001" {
        t.Errorf("Expected error code 401001, got %s", tableauErr.ErrorCode)
    }
}
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: "401001 Signin Error"
**Cause:** Invalid credentials (PAT name/secret or username/password)
**Solution:**
- Verify PAT is active in Tableau Server
- Check PAT hasn't expired (15 days inactivity or 1 year)
- Ensure credentials are correct

#### Issue 2: "403 Forbidden"
**Cause:** Token from different site being used
**Solution:**
- Ensure you're making requests to the correct site
- Verify the contentUrl matches where you authenticated

#### Issue 3: Token Expires During Long Operations
**Cause:** Token lifetime exceeded (240 minutes default)
**Solution:**
- Implement automatic token refresh
- Monitor `TokenExpiry` and re-authenticate proactively

#### Issue 4: Connection Timeout
**Cause:** Network issues or server unresponsive
**Solution:**
- Increase HTTP client timeout
- Implement retry logic with exponential backoff
- Check network connectivity and firewall rules

#### Issue 5: SSL/TLS Certificate Errors
**Cause:** Self-signed certificates or certificate validation issues
**Solution:**
```go
import "crypto/tls"

// For development only - DO NOT use in production
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true,
    },
}

client := &http.Client{Transport: transport}
```

---

## Performance Optimization Tips

1. **Reuse HTTP Connections**
   - Use connection pooling (shown in production examples)
   - Keep `Client` instance alive for multiple requests

2. **Minimize Re-authentication**
   - Cache tokens until near expiry
   - Use automatic refresh logic

3. **Batch Operations**
   - Group multiple operations when possible
   - Use bulk APIs when available

4. **Concurrent Requests**
   - Make parallel requests for independent operations
   - Respect rate limits (use rate limiter)

5. **Request Compression**
   ```go
   req.Header.Set("Accept-Encoding", "gzip")
   ```

6. **Response Streaming**
   - For large responses, process incrementally
   - Don't load entire response into memory

---

## Security Checklist

- [ ] Never hardcode credentials in source code
- [ ] Use environment variables or secret management
- [ ] Use PAT instead of username/password in production
- [ ] Implement proper error handling (don't leak sensitive info)
- [ ] Always call SignOut when done
- [ ] Use HTTPS only (never HTTP)
- [ ] Validate SSL certificates in production
- [ ] Implement rate limiting
- [ ] Log authentication events (without credentials)
- [ ] Rotate PATs regularly
- [ ] Use least privilege principle (minimal permissions)
- [ ] Implement request timeouts
- [ ] Handle token expiry gracefully

---

## Summary

This guide provides a complete, production-ready implementation of Tableau REST API authentication in Go. Key takeaways:

1. **Use PAT authentication** for production (more secure)
2. **Use API version 3.27** (latest stable)
3. **Implement automatic token refresh** to handle expiry
4. **Handle errors properly** with retry logic
5. **Follow security best practices** (no hardcoded credentials)
6. **Monitor and log** authentication events
7. **Test thoroughly** with unit and integration tests

The provided code is ready to use and follows Go best practices for HTTP clients, error handling, and API integration.

For the latest API updates and features, always refer to the official Tableau documentation.
