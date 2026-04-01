# API Standards

**This file is template-maintained.** Do not edit directly — changes will be overwritten by template updates.

## Overview

All DS app APIs follow HAL/HATEOAS patterns for consistent, discoverable, and self-documenting interfaces.

## HAL Format

### Basic Structure
All API responses follow HAL (Hypertext Application Language) format:

```json
{
  "_links": {
    "self": { "href": "/api/resource/123" }
  },
  "id": "123",
  "name": "Example Resource"
}
```

### Link Relations
- `self` — Required on all responses
- `ds:*` — Custom relations prefixed with `ds:`
- Standard IANA relations where applicable (`next`, `prev`, `collection`)

```json
{
  "_links": {
    "self": { "href": "/api/users/123" },
    "ds:projects": { "href": "/api/users/123/projects" },
    "ds:avatar": { "href": "/api/users/123/avatar" }
  },
  "id": "123",
  "email": "user@example.com"
}
```

### Embedded Resources
Use `_embedded` for included related resources:

```json
{
  "_links": {
    "self": { "href": "/api/projects/456" }
  },
  "_embedded": {
    "owner": {
      "_links": { "self": { "href": "/api/users/123" } },
      "id": "123",
      "name": "Alice"
    }
  },
  "id": "456",
  "name": "My Project"
}
```

## Discovery Endpoint

### Purpose
`GET /api/discovery` returns all available endpoints, enabling clients to navigate the API without hardcoded URLs.

### Format
```json
{
  "_links": {
    "self": { "href": "/api/discovery" },
    "ds:users": { "href": "/api/users", "title": "User management" },
    "ds:projects": { "href": "/api/projects", "title": "Project management" }
  }
}
```

### Requirement
Every route registered in `main.go` must have a corresponding link in `discovery.go`. The CI pipeline validates this.

## Error Responses

### Format
Errors use the same HAL structure with an `error` object:

```json
{
  "_links": {
    "self": { "href": "/api/resource/123" }
  },
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found"
  }
}
```

### Standard Error Codes
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid request data |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | State conflict (e.g., duplicate) |
| `INTERNAL_ERROR` | 500 | Server error |

### Validation Errors
Include field-level details for validation failures:

```json
{
  "_links": { "self": { "href": "/api/users" } },
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request data",
    "details": [
      { "field": "email", "message": "Invalid email format" },
      { "field": "name", "message": "Name is required" }
    ]
  }
}
```

## Authentication

### Endpoints
- All endpoints require authentication except:
  - `GET /api/health` — Health check
  - `GET /api/discovery` — API discovery

### Mechanism
- SSO via DSAccount
- Cookie-based authentication (`.digistratum.com` domain)
- API keys for service-to-service calls (header: `X-API-Key`)

### Response Headers
- `X-Request-Id` — Unique request identifier (for tracing)

## Collections

### Pagination
Use `page` and `limit` query parameters:

```
GET /api/users?page=2&limit=20
```

Response includes pagination links:
```json
{
  "_links": {
    "self": { "href": "/api/users?page=2&limit=20" },
    "first": { "href": "/api/users?page=1&limit=20" },
    "prev": { "href": "/api/users?page=1&limit=20" },
    "next": { "href": "/api/users?page=3&limit=20" },
    "last": { "href": "/api/users?page=5&limit=20" }
  },
  "_embedded": {
    "users": [...]
  },
  "total": 100,
  "page": 2,
  "limit": 20
}
```

### Filtering
Use query parameters for filtering:
```
GET /api/users?status=active&role=admin
```

## HTTP Methods

| Method | Purpose | Idempotent |
|--------|---------|------------|
| GET | Retrieve resource | Yes |
| POST | Create resource | No |
| PUT | Replace resource | Yes |
| PATCH | Partial update | No |
| DELETE | Remove resource | Yes |

## Related Documentation

- [Code Conventions](./conventions.md) — Handler implementation patterns
- [NFR: Security](../requirements/nfr-security.md) — Authentication details
- [Tech Stack](tech-stack.md) — Backend technologies
