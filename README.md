# External-DNS USG-DNS-API Webhook

This project implements an [external-dns](https://github.com/kubernetes-sigs/external-dns) webhook provider for [usg-dns-api](https://github.com/rclsilver-org/usg-dns-api).

## Description

This webhook allows external-dns to automatically manage DNS records on a Ubiquiti router via the usg-dns-api. It implements the external-dns webhook specification and translates operations into usg-dns-api calls.

## Features

- ✅ Retrieve existing DNS records
- ✅ Create new DNS records
- ✅ Update existing records
- ✅ Delete records
- ✅ Domain filtering
- ✅ Dry-run mode for testing
- ✅ Support for A records (IPv4)

## Prerequisites

- Go 1.23 or higher
- A running usg-dns-api instance
- An authentication token for usg-dns-api

## Installation

### Building

```bash
go build -o external-dns-usg-dns-api ./cmd/external-dns-usg-dns-api
```

Or use the Makefile (recommended):

```bash
make build
```

The Makefile automatically injects the version from Git tags. See [VERSION.md](VERSION.md) for more details.

## Configuration

Configuration is done via environment variables:

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `USG_DNS_URL` | usg-dns-api URL | Yes | - |
| `USG_DNS_TOKEN` | Authentication token | Yes | - |
| `DOMAIN_FILTER` | List of domains to manage (comma-separated) | No | All |
| `SERVER_PORT` | Webhook API listening port | No | 8888 |
| `HEALTH_PORT` | Health check listening port | No | 8080 |
| `DRY_RUN` | Test mode (no actual modifications) | No | false |

### Example

```bash
export USG_DNS_URL="http://192.168.1.1:8080"
export USG_DNS_TOKEN="your-api-token"
export DOMAIN_FILTER="example.com,test.local"
export SERVER_PORT="8888"
export HEALTH_PORT="8080"

./external-dns-usg-dns-api
```

Or with a `.env` file:

```bash
# Copy the example file
cp .env.example .env

# Edit with your values
nano .env

# Load and run
source .env
./external-dns-usg-dns-api
```

## Usage with external-dns

### Kubernetes Deployment Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: external-dns-usg-dns-api-config
  namespace: default
data:
  USG_DNS_URL: "http://192.168.1.1:8080"
  DOMAIN_FILTER: "example.com"
---
apiVersion: v1
kind: Secret
metadata:
  name: external-dns-usg-dns-api-secret
  namespace: default
type: Opaque
stringData:
  USG_DNS_TOKEN: "your-api-token"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
  namespace: default
spec:
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      serviceAccountName: external-dns
      containers:
      - name: external-dns
        image: registry.k8s.io/external-dns/external-dns:v0.15.0
        args:
        - --source=service
        - --source=ingress
        - --provider=webhook
        - --webhook-provider-url=http://localhost:8888
      - name: webhook
        image: ghcr.io/rclsilver-org/external-dns-usg-dns-api:latest
        ports:
        - containerPort: 8888
          name: http
        - containerPort: 8080
          name: health
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 5
        envFrom:
        - configMapRef:
            name: external-dns-usg-dns-api-config
        - secretRef:
            name: external-dns-usg-dns-api-secret
```

## Endpoints

The webhook exposes the following endpoints according to the external-dns specification:

### Provider endpoints (localhost:8888)

- `GET /` - Negotiation and domain filter
- `GET /records` - List all DNS records
- `POST /records` - Apply changes (create, update, delete)
- `POST /adjustendpoints` - Adjust endpoints (filtering, normalization)

### Health endpoint (0.0.0.0:8080)

- `GET /healthz` - Health check for Kubernetes

## Development

### Project Structure

```
.
├── cmd/
│   └── external-dns-usg-dns-api/
│       └── main.go                  # Entry point
├── internal/
│   ├── config/
│   │   └── config.go                # Configuration
│   ├── provider/
│   │   └── provider.go              # Provider logic
│   ├── server/
│   │   └── server.go                # HTTP server
│   ├── usgdns/
│   │   └── client.go                # usg-dns-api client
│   └── webhook/
│       └── types.go                 # external-dns types
├── go.mod
└── README.md
```

### Testing

```bash
# Dry-run mode to test without modifications
export DRY_RUN=true
./external-dns-usg-dns-api
```

## Current Limitations

- Support for A records (IPv4) only
- Single target per record
- No support for AAAA, CNAME, TXT records, etc.

## Contributing

Contributions are welcome! Feel free to open an issue or pull request.

## License

This project uses the same license as the usg-dns-api project.

## References

- [external-dns](https://github.com/kubernetes-sigs/external-dns)
- [usg-dns-api](https://github.com/rclsilver-org/usg-dns-api)
- [Webhook Provider Documentation](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/webhook-provider.md)
