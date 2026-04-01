# NFR-SEC: Security Requirements

> Security standards for DigiStratum applications.
> Based on OWASP Top 10 (2021) compliance with measurable criteria.

---

## Audit-Ready Summary

### NFR-SEC-001: OWASP Top 10 Compliance

**Acceptance Criteria:**
1. A01 - No endpoints accessible without authentication (except public routes)
2. A02 - TLS 1.2+ enforced, HttpOnly/Secure cookies
3. A03 - No raw query construction, parameterized expressions only
4. A05 - CORS restricted to explicit origins, minimal IAM permissions
5. A06 - Dependabot enabled, govulncheck and npm audit in CI
6. A07 - Centralized auth via DSAccount, rate limiting on auth endpoints
7. A09 - Structured logging with security event tracking

**Verification:**
| Method | Location |
|--------|----------|
| Security tests | `backend/internal/auth/*_test.go` |
| CI check | `govulncheck ./...`, `npm audit` |
| Manual audit | Quarterly security review |

**Evidence:** Security test results, vulnerability scan reports

---

### NFR-SEC-002: TLS 1.2+ for All Data in Transit

**Acceptance Criteria:**
1. CloudFront minimum protocol version is TLS 1.2
2. API Gateway uses AWS-managed TLS 1.2+
3. TLS 1.0 and 1.1 connection attempts are rejected
4. `curl --tlsv1.0` to app endpoints fails

**Verification:**
| Method | Location |
|--------|----------|
| CDK config | `infra/lib/cloudfront.ts` (minimumProtocolVersion) |
| Manual test | `curl --tlsv1.0 https://app.digistratum.com` (should fail) |

**Evidence:** CloudFront distribution configuration, SSL Labs scan

---

### NFR-SEC-003: Secrets in AWS Secrets Manager

**Acceptance Criteria:**
1. No secrets in source code (grep for patterns)
2. No secrets in environment variables (Lambda config audit)
3. All secrets stored in Secrets Manager with appropriate IAM access
4. Secret rotation policy defined (90 days for API keys)
5. Lambda accesses secrets via SDK, not env vars

**Verification:**
| Method | Location |
|--------|----------|
| CI check | `git-secrets` or similar scanner |
| CDK config | `infra/lib/secrets.ts` |
| Manual audit | AWS Console → Secrets Manager |

**Evidence:** No secrets in git history, Secrets Manager inventory

---

### NFR-SEC-004: Input Validation on All Endpoints

**Acceptance Criteria:**
1. All user input validated before processing
2. Validation rejects: empty required fields, oversized inputs, invalid formats
3. XSS patterns rejected in text inputs
4. UUID fields validated with regex
5. Validation errors return 400 with field-specific messages

**Verification:**
| Method | Location |
|--------|----------|
| Unit tests | `backend/internal/api/*_test.go` |
| Integration tests | `backend/test/integration/validation_test.go` |

**Evidence:** CI test results showing validation test coverage

---

### NFR-SEC-005: CORS Restricted to Allowed Origins

**Acceptance Criteria:**
1. Production CORS allows only `*.digistratum.com` origins
2. Wildcard (`*`) never used with credentials
3. CORS preflight returns appropriate headers
4. Requests from unauthorized origins receive no CORS headers
5. Allowed methods limited to what's actually needed

**Verification:**
| Method | Location |
|--------|----------|
| CDK config | `infra/lib/api.ts` (corsPreflight) |
| Unit test | `backend/internal/middleware/cors_test.go` |
| Manual test | `curl -H "Origin: https://evil.com"` shows no CORS headers |

**Evidence:** API Gateway configuration, CORS header tests

---

## Quick Reference

| Requirement | Description | Status |
|-------------|-------------|--------|
| NFR-SEC-001 | OWASP Top 10 compliance | Required |
| NFR-SEC-002 | All data encrypted in transit (TLS 1.2+) | Required |
| NFR-SEC-003 | Secrets stored in AWS Secrets Manager | Required |
| NFR-SEC-004 | Input validation on all endpoints | Required |
| NFR-SEC-005 | CORS configured for allowed origins only | Required |

---

## OWASP Top 10 Compliance

### A01:2021 – Broken Access Control

**Risk:** Users acting outside their intended permissions.

**Requirements:**
- ✅ Deny by default for all API endpoints
- ✅ Authentication required via DSAccount SSO
- ✅ Tenant isolation enforced at data layer
- ✅ All DynamoDB queries scoped to tenant partition key
- ✅ Rate limiting on authentication endpoints

**Implementation:**
```go
// backend/internal/auth/middleware.go
func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, err := validateSession(r)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        // Inject session into context for tenant-scoped queries
        ctx := context.WithValue(r.Context(), SessionKey, session)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// All data access uses tenant from session
func (r *Repository) GetItems(ctx context.Context) ([]Item, error) {
    session := ctx.Value(SessionKey).(*Session)
    return r.queryByTenant(session.TenantID)
}
```

**Testing:**
```go
func TestUnauthorizedAccess(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/items", nil)
    // No auth cookie
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCrossTenantAccess(t *testing.T) {
    // User from tenant A should not access tenant B data
    req := httptest.NewRequest("GET", "/api/items/tenant-b-item", nil)
    req = req.WithContext(contextWithTenant(ctx, "tenant-a"))
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    assert.Equal(t, http.StatusNotFound, w.Code)
}
```

### A02:2021 – Cryptographic Failures

**Risk:** Exposure of sensitive data due to weak or missing cryptography.

**Requirements:**
- ✅ TLS 1.2+ enforced via CloudFront
- ✅ HTTPS-only cookies in production
- ✅ Secrets stored in AWS Secrets Manager (never in code)
- ✅ JWT tokens validated with proper signature verification
- ✅ DynamoDB encryption at rest (AWS managed keys)

**Cookie Configuration:**
```go
http.SetCookie(w, &http.Cookie{
    Name:     "ds_session",
    Value:    sessionToken,
    HttpOnly: true,                    // Prevent XSS access
    Secure:   env != "development",    // HTTPS only in prod
    SameSite: http.SameSiteLaxMode,    // CSRF protection
    Path:     "/",
    MaxAge:   86400,                   // 24 hours
})
```

### A03:2021 – Injection

**Risk:** SQL, NoSQL, OS, or LDAP injection attacks.

**Requirements:**
- ✅ DynamoDB uses parameterized expressions (no raw queries)
- ✅ No SQL databases in architecture
- ✅ Input validation on all user-provided data
- ✅ No shell command execution from user input

**Safe Query Pattern:**
```go
// Always use expression builders - never string concatenation
input := &dynamodb.QueryInput{
    TableName:              aws.String(tableName),
    KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
    ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
        ":pk": {S: aws.String(tenantID)},      // Parameterized
        ":sk": {S: aws.String("ITEM#")},       // Parameterized
    },
}
```

### A04:2021 – Insecure Design

**Risk:** Missing or ineffective control design.

**Requirements:**
- ✅ Security requirements defined in REQUIREMENTS.md
- ✅ Threat modeling for authentication flows
- ✅ Secure defaults (deny by default, fail closed)
- ✅ Multi-tenant isolation by design

**Design Principles:**
1. Defense in depth (multiple security layers)
2. Least privilege (minimal permissions)
3. Fail secure (errors deny access, not grant)
4. Separation of duties (auth vs. data access)

### A05:2021 – Security Misconfiguration

**Risk:** Missing or incorrect security hardening.

**Requirements:**
- ✅ CDK infrastructure as code (repeatable, auditable)
- ✅ Minimal IAM permissions for Lambda
- ✅ CloudFront security headers configured
- ✅ No debug information exposed in production
- ✅ CORS restricted to allowed origins

**CDK Security Configuration:**
```typescript
// Minimal Lambda permissions
const lambdaRole = new iam.Role(this, 'LambdaRole', {
    assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
    managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName(
            'service-role/AWSLambdaBasicExecutionRole'
        ),
    ],
});

// Only the specific table, not dynamodb:*
table.grantReadWriteData(lambdaRole);
```

### A06:2021 – Vulnerable and Outdated Components

**Risk:** Known vulnerabilities in dependencies.

**Requirements:**
- ✅ Dependabot enabled for automated updates
- ✅ npm audit in CI pipeline
- ✅ govulncheck for Go vulnerabilities
- ✅ Regular dependency updates (at least monthly)

**CI Gate:**
```yaml
- name: Go vulnerability check
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...

- name: NPM audit
  run: npm audit --audit-level=high
```

### A07:2021 – Identification and Authentication Failures

**Risk:** Weak authentication implementation.

**Requirements:**
- ✅ Centralized authentication via DSAccount SSO
- ✅ No custom password storage
- ✅ Session tokens with expiration
- ✅ HttpOnly cookies prevent token theft
- ✅ Rate limiting on auth endpoints

### A08:2021 – Software and Data Integrity Failures

**Risk:** Code and infrastructure without integrity verification.

**Requirements:**
- ✅ Signed commits recommended
- ✅ Branch protection on main
- ✅ CI/CD pipeline validates all changes
- ✅ CDK diff review before deployment
- ✅ Canary deployment with automatic rollback

### A09:2021 – Security Logging and Monitoring Failures

**Risk:** Insufficient logging for security events.

**Requirements:**
- ✅ Structured logging to CloudWatch
- ✅ Request correlation IDs
- ✅ Authentication event logging
- ✅ Security alerts on anomalies

**Logging Pattern:**
```go
// Log security events without sensitive data
log.Info("auth_event",
    "event", "login_success",
    "user_id", session.UserID,
    "tenant_id", session.TenantID,
    "correlation_id", r.Header.Get("X-Request-ID"),
    // Never log: tokens, passwords, full session data
)
```

### A10:2021 – Server-Side Request Forgery (SSRF)

**Risk:** Server makes requests to unintended locations.

**Requirements:**
- ✅ No user-controlled URLs in server requests
- ✅ DSAccount SSO URL hardcoded in configuration
- ✅ Lambda has no VPC access by default (isolated)
- ✅ Outbound requests only to known services

---

## TLS Requirements (NFR-SEC-002)

**Target:** All data encrypted in transit using TLS 1.2 or higher.

### Configuration

| Component | TLS Version | Configuration |
|-----------|-------------|---------------|
| CloudFront | TLS 1.2+ | ViewerProtocolPolicy.HTTPS_ONLY |
| API Gateway | TLS 1.2+ | Default (AWS managed) |
| Lambda → DynamoDB | TLS 1.2+ | AWS SDK (automatic) |
| DSAccount SSO | TLS 1.2+ | HTTPS only |

### CDK Implementation

```typescript
// CloudFront distribution
const distribution = new cloudfront.Distribution(this, 'Distribution', {
    defaultBehavior: {
        origin: new origins.S3Origin(bucket),
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
    },
    minimumProtocolVersion: cloudfront.SecurityPolicyProtocol.TLS_V1_2_2021,
});
```

### Testing

```bash
# Verify TLS version
curl -svo /dev/null https://app.example.com 2>&1 | grep "TLS"

# Check for TLS 1.0/1.1 (should fail)
curl --tlsv1.0 https://app.example.com  # Should fail
curl --tlsv1.1 https://app.example.com  # Should fail
```

---

## Secrets Management (NFR-SEC-003)

**Target:** All secrets stored in AWS Secrets Manager, never in code or environment variables.

### Secret Categories

| Secret Type | Storage | Rotation |
|-------------|---------|----------|
| API Keys | Secrets Manager | 90 days |
| Database credentials | Secrets Manager | 30 days |
| JWT signing keys | Secrets Manager | 180 days |
| Third-party tokens | Secrets Manager | Per vendor policy |

### Implementation

```typescript
// CDK secret creation
const appSecret = new secretsmanager.Secret(this, 'AppSecret', {
    secretName: `/${appName}/${environment}/secrets`,
    generateSecretString: {
        secretStringTemplate: JSON.stringify({
            DSACCOUNT_APP_SECRET: '',  // Injected post-deploy
        }),
        generateStringKey: 'INTERNAL_API_KEY',
    },
});

// Grant Lambda read access
appSecret.grantRead(lambdaFunction);
```

### Lambda Access Pattern

```go
func getSecret(ctx context.Context, secretName string) (string, error) {
    cfg, _ := config.LoadDefaultConfig(ctx)
    client := secretsmanager.NewFromConfig(cfg)
    
    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretName),
    })
    if err != nil {
        return "", err
    }
    return *result.SecretString, nil
}
```

### Checklist

- [ ] No secrets in source code
- [ ] No secrets in environment variables (use Secrets Manager references)
- [ ] No secrets in CI/CD logs
- [ ] Secrets have appropriate IAM access controls
- [ ] Rotation policy defined for each secret

---

## Input Validation (NFR-SEC-004)

**Target:** All user input validated before processing.

### Validation Principles

1. **Validate on the server** - Never trust client-side validation alone
2. **Allowlist over blocklist** - Define what IS allowed, not what isn't
3. **Validate early** - Check inputs before any processing
4. **Fail fast** - Reject invalid input immediately
5. **Type coercion** - Convert to expected types safely

### Go Backend Validation

```go
package validation

import (
    "regexp"
    "unicode/utf8"
)

func MaxLength(s string, max int) bool {
    return utf8.RuneCountInString(s) <= max
}

func MinLength(s string, min int) bool {
    return utf8.RuneCountInString(s) >= min
}

var safeStringPattern = regexp.MustCompile(`^[\p{L}\p{N}\s\-_.,!?@#$%&*()[\]{}:;"']+$`)
func SafeString(s string) bool {
    return safeStringPattern.MatchString(s)
}

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
func Email(s string) bool {
    return emailPattern.MatchString(s) && len(s) <= 254
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
func UUID(s string) bool {
    return uuidPattern.MatchString(strings.ToLower(s))
}
```

### Request Validation Example

```go
type CreateItemRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Priority    int    `json:"priority"`
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
    var req CreateItemRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    errors := make(map[string]string)
    
    req.Name = strings.TrimSpace(req.Name)
    if !validation.MinLength(req.Name, 1) {
        errors["name"] = "Name is required"
    } else if !validation.MaxLength(req.Name, 100) {
        errors["name"] = "Name must be 100 characters or less"
    } else if !validation.SafeString(req.Name) {
        errors["name"] = "Name contains invalid characters"
    }
    
    if len(errors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error":  "Validation failed",
            "fields": errors,
        })
        return
    }
    // Proceed with validated data...
}
```

### TypeScript Frontend Validation

```typescript
export const validators = {
    required: (value: string): boolean => value.trim().length > 0,
    maxLength: (value: string, max: number): boolean => value.length <= max,
    email: (value: string): boolean => {
        const pattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
        return pattern.test(value) && value.length <= 254;
    },
    uuid: (value: string): boolean => {
        const pattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
        return pattern.test(value);
    },
    safeString: (value: string): boolean => {
        const dangerous = /<script|javascript:|on\w+=/i;
        return !dangerous.test(value);
    }
};
```

### Testing

```go
func TestInputValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected int
    }{
        {"empty name", `{"name":""}`, 400},
        {"too long", `{"name":"` + strings.Repeat("a", 101) + `"}`, 400},
        {"XSS attempt", `{"name":"<script>alert(1)</script>"}`, 400},
        {"valid", `{"name":"Valid Name"}`, 201},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("POST", "/api/items", strings.NewReader(tt.input))
            w := httptest.NewRecorder()
            handler.ServeHTTP(w, req)
            assert.Equal(t, tt.expected, w.Code)
        })
    }
}
```

---

## CORS Configuration (NFR-SEC-005)

**Target:** CORS restricted to explicitly allowed origins only.

### Allowed Origins by Environment

| Environment | Allowed Origins |
|-------------|-----------------|
| Production | `https://app.digistratum.com`, `https://www.digistratum.com` |
| Staging | `https://staging.digistratum.com` |
| Development | `http://localhost:5173`, `http://localhost:3000` |

### CDK Configuration

```typescript
const httpApi = new apigwv2.HttpApi(this, 'HttpApi', {
    corsPreflight: {
        allowOrigins: getAllowedOrigins(props.environment),
        allowMethods: [
            apigwv2.CorsHttpMethod.GET,
            apigwv2.CorsHttpMethod.POST,
            apigwv2.CorsHttpMethod.PUT,
            apigwv2.CorsHttpMethod.DELETE,
            apigwv2.CorsHttpMethod.OPTIONS,
        ],
        allowHeaders: [
            'Content-Type',
            'Authorization',
            'X-Tenant-ID',
            'X-Request-ID',
        ],
        allowCredentials: true,
        maxAge: cdk.Duration.hours(1),
    },
});
```

### Backend Middleware

```go
func CORS(next http.Handler) http.Handler {
    env := os.Getenv("ENVIRONMENT")
    origins := allowedOrigins[env]
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        allowed := false
        for _, o := range origins {
            if origin == o {
                allowed = true
                break
            }
        }
        
        if allowed {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-Request-ID")
        }
        
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### CORS Security Checklist

- [ ] Never use `Access-Control-Allow-Origin: *` with credentials
- [ ] Explicitly list allowed origins (no wildcards in production)
- [ ] Limit allowed methods to what's actually needed
- [ ] Limit allowed headers to what's actually needed
- [ ] Validate Origin header on server side

---

## Content Security Policy

### CloudFront Response Headers

```typescript
const securityHeadersPolicy = new cloudfront.ResponseHeadersPolicy(
    this, 'SecurityHeaders', {
        securityHeadersBehavior: {
            contentSecurityPolicy: {
                contentSecurityPolicy: buildCSP(props.environment),
                override: true,
            },
            contentTypeOptions: { override: true },
            frameOptions: {
                frameOption: cloudfront.HeadersFrameOption.DENY,
                override: true,
            },
            strictTransportSecurity: {
                accessControlMaxAge: cdk.Duration.days(365),
                includeSubdomains: true,
                preload: true,
                override: true,
            },
        },
    }
);

function buildCSP(env: string): string {
    const directives = [
        "default-src 'self'",
        env === 'dev' 
            ? "script-src 'self' 'unsafe-inline' 'unsafe-eval'"
            : "script-src 'self'",
        "style-src 'self' 'unsafe-inline'",
        "img-src 'self' data: https:",
        "font-src 'self'",
        `connect-src 'self' https://api.digistratum.com`,
        "frame-src 'none'",
        "form-action 'self'",
        "base-uri 'self'",
        "object-src 'none'",
        ...(env !== 'dev' ? ["upgrade-insecure-requests"] : []),
    ];
    return directives.join('; ');
}
```

---

## Security Checklist

### Pre-Deployment

- [ ] No secrets in code or environment variables
- [ ] All inputs validated
- [ ] Authentication required on all non-public endpoints
- [ ] CORS configured correctly
- [ ] npm audit / govulncheck passes
- [ ] CSP headers configured
- [ ] HTTPS enforced

### Periodic Review

- [ ] Dependency updates applied (monthly)
- [ ] Secret rotation completed (per policy)
- [ ] Security logs reviewed
- [ ] Access patterns audited
- [ ] Penetration testing (annually)

---

## Traceability

| Requirement | Implementation | Test |
|-------------|----------------|------|
| NFR-SEC-001 | OWASP controls | Security test suite |
| NFR-SEC-002 | CloudFront TLS config | TLS version tests |
| NFR-SEC-003 | Secrets Manager | No secrets in code |
| NFR-SEC-004 | Validation middleware | Input validation tests |
| NFR-SEC-005 | CORS middleware | CORS header tests |

---

*Last updated: 2026-03-22*
