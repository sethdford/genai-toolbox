# Tableau REST API - Quick Reference Card

## Authentication Endpoints

```
POST /api/{version}/auth/signin   - Authenticate and get token
POST /api/{version}/auth/signout  - Invalidate token
POST /api/{version}/auth/switchSite - Switch to different site
```

## Current API Version

**Latest Stable:** `3.27` (Tableau 2025.3)

## Base URL Format

```
https://{server}/api/{version}/{endpoint}

Examples:
https://tableau.example.com/api/3.27/auth/signin
https://10ay.online.tableau.com/api/3.27/sites/{site-id}/workbooks
```

## Authentication Methods

### Personal Access Token (PAT) - RECOMMENDED

**JSON Request:**
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

**XML Request:**
```xml
<tsRequest>
  <credentials personalAccessTokenName="TOKEN_NAME"
               personalAccessTokenSecret="TOKEN_SECRET">
    <site contentUrl="SITE_NAME" />
  </credentials>
</tsRequest>
```

**Go Code:**
```go
client.SignInWithPAT("TOKEN_NAME", "TOKEN_SECRET", "SITE_NAME")
```

### Username/Password

**JSON Request:**
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

**XML Request:**
```xml
<tsRequest>
  <credentials name="USERNAME" password="PASSWORD">
    <site contentUrl="SITE_NAME" />
  </credentials>
</tsRequest>
```

**Go Code:**
```go
client.SignInWithPassword("USERNAME", "PASSWORD", "SITE_NAME")
```

## Response Format

**JSON Response:**
```json
{
  "credentials": {
    "token": "AUTH_TOKEN_HERE",
    "estimatedTimeToExpiration": "239:59:59",
    "site": {
      "id": "SITE_ID",
      "contentUrl": "SITE_NAME"
    },
    "user": {
      "id": "USER_ID"
    }
  }
}
```

**Extract:**
- `token` → Use in `X-Tableau-Auth` header
- `site.id` → Use in API endpoints
- `user.id` → Authenticated user

## Required Headers

### Sign In Request
```
Content-Type: application/json
Accept: application/json
```

### Authenticated Requests
```
X-Tableau-Auth: {token}
Content-Type: application/json
Accept: application/json
```

## Token Lifecycle

```
1. Sign In → Get Token (240 min lifetime)
2. Include token in X-Tableau-Auth header
3. Monitor expiry (proactive refresh at 235 min)
4. Sign Out → Invalidate token
```

## Common Error Codes

| Code   | HTTP | Description |
|--------|------|-------------|
| 401000 | 401  | No auth credentials |
| 401001 | 401  | Login error / invalid PAT |
| 401002 | 401  | Invalid credentials |
| 401003 | 401  | Switch site error |
| 403000 | 403  | Forbidden (wrong site) |
| 404000 | 404  | Resource not found |
| 400000 | 400  | Bad request |

## Go Quick Start

### 1. Create Client
```go
client := tableau.NewClient(tableau.ClientConfig{
    ServerURL:  "https://tableau.example.com",
    APIVersion: "3.27",
    Timeout:    30 * time.Second,
})
```

### 2. Authenticate
```go
// PAT (recommended)
err := client.SignInWithPAT(
    os.Getenv("TABLEAU_PAT_NAME"),
    os.Getenv("TABLEAU_PAT_SECRET"),
    "", // Empty for default site
)

// Username/Password
err := client.SignInWithPassword(
    "user@example.com",
    "password",
    "", // Empty for default site
)
```

### 3. Make Requests
```go
resp, err := client.DoRequest(
    "GET",
    "/sites/"+client.SiteID+"/workbooks",
    nil,
)
defer resp.Body.Close()
```

### 4. Clean Up
```go
defer client.SignOut()
```

## Common Endpoints

```go
// List workbooks
GET /sites/{site-id}/workbooks

// Get workbook
GET /sites/{site-id}/workbooks/{workbook-id}

// List projects
GET /sites/{site-id}/projects

// List users
GET /sites/{site-id}/users

// Get site info
GET /sites/{site-id}

// Query views
GET /sites/{site-id}/views

// List data sources
GET /sites/{site-id}/datasources
```

## Environment Variables

```bash
# Required
export TABLEAU_SERVER="https://tableau.example.com"
export TABLEAU_PAT_NAME="your-token-name"
export TABLEAU_PAT_SECRET="your-token-secret"

# Optional
export TABLEAU_SITE=""           # Empty for default site
export TABLEAU_API_VERSION="3.27"
```

## Error Handling Pattern

```go
err := client.SignInWithPAT(tokenName, tokenSecret, site)
if err != nil {
    if tableau.IsAuthError(err) {
        // Handle auth error
    }

    if tableauErr, ok := err.(*tableau.TableauError); ok {
        log.Printf("Code: %s", tableauErr.ErrorCode)
        log.Printf("Summary: %s", tableauErr.Summary)
        log.Printf("Detail: %s", tableauErr.Detail)
    }
}
```

## Token Management

```go
// Check if authenticated
if client.IsAuthenticated() {
    // Token is valid
}

// Check if near expiry
if client.IsTokenExpired() {
    // Refresh needed
}

// Auto-refresh
creds := tableau.AuthCredentials{
    Type:           "pat",
    TokenName:      os.Getenv("TABLEAU_PAT_NAME"),
    TokenSecret:    os.Getenv("TABLEAU_PAT_SECRET"),
    SiteContentUrl: os.Getenv("TABLEAU_SITE"),
}
client.EnsureAuthenticated(creds)
```

## Retry Pattern

```go
maxRetries := 3
for attempt := 0; attempt < maxRetries; attempt++ {
    err := client.SignInWithPAT(tokenName, tokenSecret, site)
    if err == nil {
        break
    }

    // Don't retry invalid credentials
    if tableauErr, ok := err.(*tableau.TableauError); ok {
        if tableauErr.ErrorCode == tableau.ErrorInvalidCredentials {
            return err
        }
    }

    // Exponential backoff: 1s, 2s, 4s
    backoff := time.Duration(1<<uint(attempt)) * time.Second
    time.Sleep(backoff)
}
```

## cURL Examples

### Sign In (PAT)
```bash
curl -X POST \
  "https://tableau.example.com/api/3.27/auth/signin" \
  -H "Content-Type: application/json" \
  -d '{
    "credentials": {
      "personalAccessTokenName": "TOKEN_NAME",
      "personalAccessTokenSecret": "TOKEN_SECRET",
      "site": {"contentUrl": ""}
    }
  }'
```

### List Workbooks
```bash
curl -X GET \
  "https://tableau.example.com/api/3.27/sites/{site-id}/workbooks" \
  -H "X-Tableau-Auth: {token}"
```

### Sign Out
```bash
curl -X POST \
  "https://tableau.example.com/api/3.27/auth/signout" \
  -H "X-Tableau-Auth: {token}"
```

## Best Practices Checklist

- [ ] Use PAT instead of username/password in production
- [ ] Store credentials in environment variables (never hardcode)
- [ ] Always call SignOut() when done (use defer)
- [ ] Implement token refresh for long-running apps
- [ ] Handle errors properly (check error codes)
- [ ] Use latest API version (3.27)
- [ ] Set appropriate timeouts
- [ ] Log authentication events (without credentials)
- [ ] Use HTTPS only (never HTTP)
- [ ] Validate SSL certificates in production

## PAT Limitations

- **Concurrent Sessions:** Cannot use same PAT in multiple sessions
- **Inactivity Expiry:** 15 days of no use
- **Max Lifetime:** 1 year (Tableau Server)
- **Per User Limit:** Maximum 10 PATs per user
- **MFA:** Required for Tableau Cloud with MFA enabled

## Token Lifetime

| Setting | Default | Configurable |
|---------|---------|--------------|
| Token Duration | 240 minutes | Yes (server config) |
| Inactivity Timeout | 240 minutes | Yes (server config) |
| PAT Inactivity | 15 days | No |
| PAT Max Age | 1 year | No |

## URL Encoding

```go
import "net/url"

// Encode site content URL if it contains special characters
siteURL := url.PathEscape("my-site-name")
```

## Testing Checklist

```bash
# Test authentication
curl -X POST "https://tableau.example.com/api/3.27/auth/signin" \
  -H "Content-Type: application/json" \
  -d '{"credentials":{"personalAccessTokenName":"test","personalAccessTokenSecret":"test","site":{"contentUrl":""}}}'

# Test with invalid credentials (should return 401001)
# Test with wrong site (should return 403)
# Test token expiry (after 240 minutes)
# Test concurrent PAT usage (should fail)
```

## Documentation Links

| Resource | URL |
|----------|-----|
| REST API Reference | https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_ref.htm |
| Authentication | https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_auth.htm |
| Error Handling | https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_errors.htm |
| PAT Documentation | https://help.tableau.com/current/server/en-us/security_personal_access_tokens.htm |
| API Versions | https://help.tableau.com/current/api/rest_api/en-us/REST/rest_api_concepts_versions.htm |
| Postman Collection | https://github.com/tableau/tableau-postman |

## Troubleshooting Quick Guide

| Symptom | Cause | Solution |
|---------|-------|----------|
| 401001 error | Invalid PAT | Check PAT exists and is active |
| 403 error | Wrong site | Verify contentUrl matches |
| Token expired | 240 min passed | Re-authenticate |
| Connection timeout | Network/slow server | Increase timeout |
| SSL error | Self-signed cert | Install CA cert (prod) or skip verify (dev only) |
| Concurrent error | Same PAT in 2+ sessions | Use different PATs |

## Complete Minimal Example

```go
package main

import (
    "log"
    "os"
    "yourproject/tableau"
)

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

    log.Printf("Authenticated! Site: %s", client.SiteID)
}
```

---

**Pro Tip:** Always use the latest API version and PAT authentication for production deployments.
