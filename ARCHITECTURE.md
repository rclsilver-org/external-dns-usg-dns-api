# Architecture

This document describes the architecture of external-dns-usg-dns-api.

## Overview

```
┌─────────────────────────────────────────────────────┐
│                Kubernetes Cluster                   │
│                                                     │
│  ┌─────────────────────────────────────────────┐    │
│  │              External-DNS Pod               │    │
│  │                                             │    │
│  │  ┌─────────────────┐  ┌──────────────────┐  │    │
│  │  │  external-dns   │  │   webhook        │  │    │
│  │  │   (container)   │──│   (container)    │  │    │
│  │  │                 │  │                  │  │    │
│  │  │  - Watches      │  │  ┌─────────────┐ │  │    │
│  │  │    Services     │  │  │ API Server  │ │  │    │
│  │  │  - Watches      │  │  │ Port: 8888  │ │  │    │
│  │  │    Ingress      │  │  │             │ │  │    │
│  │  │  - Webhook      │  │  │ - Negotiate │ │  │    │
│  │  │    provider     │──│─▶│ - Records   │ │  │    │
│  │  │                 │  │  │ - Adjust    │ │  │    │
│  │  │                 │  │  └─────────────┘ │  │    │
│  │  │                 │  │                  │  │    │
│  │  │                 │  │  ┌─────────────┐ │  │    │
│  │  │                 │  │  │ Health Srv  │ │  │    │
│  │  │   Health        │──│─▶│ Port: 8080  │ │  │    │
│  │  │   Probes        │  │  │             │ │  │    │
│  │  │                 │  │  │ - /healthz  │ │  │    │
│  │  │                 │  │  │ - /readyz   │ │  │    │
│  │  │                 │  │  │ - /livez    │ │  │    │
│  │  │                 │  │  └─────────────┘ │  │    │
│  │  └─────────────────┘  └──────────────────┘  │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
                          │
                          │ HTTP
                          ▼
            ┌───────────────────────────┐
            │   USG DNS API             │
            │   (Ubiquiti Router)       │
            │                           │
            │   - DNS Management        │
            │   - Port 8080             │
            │   - Auth: Token           │
            └───────────────────────────┘
```

## Components

### 1. External-DNS (main container)

External-DNS watches Kubernetes resources (Services, Ingress) and detects changes that require DNS modifications.

**Responsibilities:**
- Watch Kubernetes resources
- Detect annotation changes
- Calculate necessary DNS changes
- Call the webhook provider

### 2. Webhook Provider (our implementation)

The webhook implements the external-dns interface and communicates with the usg-dns-api.

**Architecture:**
- Two independent HTTP servers for better security and isolation
- Main API server (port 8888) handles webhook requests from external-dns
- Health server (port 8080) handles Kubernetes probes

**Responsibilities:**
- Expose webhook API according to external-dns specification
- Translate requests into usg-dns-api calls
- Handle DNS record CRUD operations
- Filter by domain
- Provide health endpoints for Kubernetes probes

**Endpoints:**

*API Server (port 8888):*
- `GET /` - Negotiate domain filter
- `GET /records` - List DNS records
- `POST /records` - Apply DNS changes
- `POST /adjustendpoints` - Adjust endpoints before processing

*Health Server (port 8080):*
- `GET /healthz` - Health check (liveness probe)
- `GET /readyz` - Ready check (readiness probe)
- `GET /livez` - Liveness check

This separation allows:
- External-DNS to communicate only with the API server
- Kubernetes to monitor health without exposing the API
- Better security through port segregation
- Cleaner separation of concerns

### 3. USG DNS API

The API on the Ubiquiti router that actually manages DNS records.

**Responsibilities:**
- Store DNS records
- Generate hosts file
- Handle token authentication

## Data Flow

### 1. Record Creation

```
Service/Ingress created
       │
       ▼
external-dns detects
       │
       ▼
POST /records (with changes)
       │
       ▼
webhook provider
       │
       ▼
POST /records (usg-dns-api)
       │
       ▼
Record created
```

### 2. Record Update

```
Service/Ingress modified
       │
       ▼
external-dns detects
       │
       ▼
POST /records (with updateOld/updateNew)
       │
       ▼
webhook provider (finds ID)
       │
       ▼
PUT /records/:id (usg-dns-api)
       │
       ▼
Record updated
```

### 3. Record Deletion

```
Service/Ingress deleted
       │
       ▼
external-dns detects
       │
       ▼
POST /records (with delete)
       │
       ▼
webhook provider (finds ID)
       │
       ▼
DELETE /records/:id (usg-dns-api)
       │
       ▼
Record deleted
```

## Code Structure

```
external-dns-usg-dns-api/
├── cmd/
│   └── external-dns-usg-dns-api/
│       └── main.go                    # Entry point
│
├── internal/
│   ├── config/
│   │   └── config.go                  # Configuration loading
│   │
│   ├── webhook/
│   │   └── types.go                   # external-dns types
│   │
│   ├── usgdns/
│   │   └── client.go                  # HTTP client for usg-dns-api
│   │
│   ├── provider/
│   │   ├── provider.go                # Business logic
│   │   └── provider_test.go           # Unit tests
│   │
│   └── server/
│       └── server.go                  # HTTP webhook server
│
├── go.mod
├── Makefile
└── README.md
```

## Webhook Endpoints

### Provider endpoints (localhost:8888)

#### `GET /`
**Negotiation and domain filter**

Returns the configured domain filter.

```json
{
  "filters": ["example.com"]
}
```

#### `GET /records`
**List records**

Returns all current DNS records.

```json
[
  {
    "dnsName": "test.example.com",
    "targets": ["192.168.1.100"],
    "recordType": "A",
    "recordTTL": 300
  }
]
```

#### `POST /records`
**Apply changes**

Receives changes to apply.

```json
{
  "create": [...],
  "updateOld": [...],
  "updateNew": [...],
  "delete": [...]
}
```

#### `POST /adjustendpoints`
**Adjust endpoints**

Allows filtering and normalizing endpoints before processing.

### Health endpoint (0.0.0.0:8080)

#### `GET /healthz`
**Health check**

Returns 200 OK if the service is operational.

## Error Handling

### HTTP Status Codes

- `200 OK`: Success
- `204 No Content`: Success (no content)
- `400 Bad Request`: Invalid request
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

### Retry

External-dns only retries 5xx errors. 4xx errors are considered final.

## Configuration

### Environment Variables

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `USG_DNS_URL` | string | Yes | - | usg-dns-api URL |
| `USG_DNS_TOKEN` | string | Yes | - | Authentication token |
| `DOMAIN_FILTER` | string | No | - | Domains to manage (comma-separated) |
| `SERVER_PORT` | int | No | 8888 | Webhook port |
| `DRY_RUN` | bool | No | false | Test mode |

## Limitations

### Technical

1. **Record Types**: Only A records are supported
2. **Multiple Targets**: Single target per record
3. **TTL**: TTL is fixed (300s) as not supported by usg-dns-api
4. **Concurrency**: No lock management (last write wins)

### Functional

1. **No DNSSEC**
2. **No secondary zones**
3. **No wildcard DNS**

## Performance

### Possible Optimizations

1. **Local Cache**: Cache records to avoid repeated API calls
2. **Batch Operations**: Group operations if possible
3. **Keep-alive Connection**: Reuse HTTP connections

### Metrics

The webhook doesn't expose Prometheus metrics yet, but this could be added:

- Number of requests per endpoint
- usg-dns-api response time
- Number of errors
- Number of managed records

## Security

### Communication

- Webhook listens on localhost:8888 (not exposed)
- Health check listens on 0.0.0.0:8080 (exposed)
- Communication with usg-dns-api via HTTP token

### Recommendations

1. **Token Rotation**: Regularly change the usg-dns-api token
2. **TLS**: Use HTTPS for usg-dns-api if possible
3. **Network Policies**: Limit pod network access
4. **RBAC**: Give only necessary permissions to external-dns

## Testing

### Unit Tests

```bash
make test
```

### Integration Tests

Use dry-run mode to test without modifying DNS:

```bash
export DRY_RUN=true
./external-dns-usg-dns-api
```

### Manual Tests

```bash
# Negotiation
curl http://localhost:8888/

# List records
curl http://localhost:8888/records

# Test creation
curl -X POST http://localhost:8888/records \
  -H "Content-Type: application/external.dns.webhook+json;version=1" \
  -d '{"create":[{"dnsName":"test.example.com","targets":["1.2.3.4"],"recordType":"A"}]}'
```

## Future Enhancements

### Short Term

- [ ] Prometheus metrics
- [ ] Local record caching
- [ ] Automated integration tests

### Long Term

- [ ] Webhook events to notify changes
- [ ] Monitoring interface
