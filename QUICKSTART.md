# Quick Start Guide

This guide will help you get started quickly with external-dns-usg-dns-api.

## Prerequisites

1. A running [usg-dns-api](https://github.com/rclsilver-org/usg-dns-api) instance
2. An authentication token generated with `usg-dns-api generate-token`
3. Go 1.23+ installed (for building)

## Local Testing

### 1. Configuration

Copy the example file and configure your variables:

```bash
cp .env.example .env
nano .env
```

Configure at minimum:
- `USG_DNS_URL`: Your API URL (e.g., `http://192.168.1.1:8080`)
- `USG_DNS_TOKEN`: Your authentication token

### 2. Building

```bash
make build
# or
go build -o external-dns-usg-dns-api ./cmd/external-dns-usg-dns-api
```

### 3. Running

```bash
# Load environment variables
source .env

# Run the application
./external-dns-usg-dns-api
```

Or use the test script:

```bash
source .env
./test-local.sh
```

### 4. Manual Testing

In another terminal, test the endpoints:

```bash
# Negotiation and domain filter
curl http://localhost:8888/

# List records
curl http://localhost:8888/records

# Health check
curl http://localhost:8080/healthz
```

## Kubernetes Deployment

### 1. Prepare the manifest

Edit `k8s-deployment.yaml` and configure:

```yaml
data:
  USG_DNS_URL: "http://192.168.1.1:8080"  # Your URL
  DOMAIN_FILTER: "example.com"             # Your domain

stringData:
  USG_DNS_TOKEN: "your-token-here"         # Your token
```

### 2. Deploy

```bash
kubectl apply -f k8s-deployment.yaml
```

### 3. Verify

```bash
# Check pods
kubectl get pods -l app=external-dns

# View logs
kubectl logs -l app=external-dns -c webhook -f
kubectl logs -l app=external-dns -c external-dns -f
```

## Testing with a Service

Create a test service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: test-service
  annotations:
    external-dns.alpha.kubernetes.io/hostname: test.example.com
spec:
  type: LoadBalancer
  loadBalancerIP: 192.168.1.100
  ports:
  - port: 80
  selector:
    app: test
```

After a few moments, you should see the DNS record created on your Ubiquiti router.

## Testing with an Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service
            port:
              number: 80
```

## Dry-Run Mode

To test without applying changes:

### Locally

```bash
export DRY_RUN=true
./external-dns-usg-dns-api
```

### In Kubernetes

Add the argument to external-dns:

```yaml
args:
- --dry-run=true
```

And/or configure the webhook:

```yaml
data:
  DRY_RUN: "true"
```

## Troubleshooting

### Webhook won't start

Check environment variables:

```bash
kubectl logs -l app=external-dns -c webhook
```

### Records are not being created

1. Check that the domain matches the filter
2. Check external-dns logs
3. Manually test the usg-dns-api

```bash
curl -H "Authorization: $USG_DNS_TOKEN" $USG_DNS_URL/records
```

### API connection error

Check:
- The URL is correct and accessible
- The token is valid
- The firewall allows the connection

## Useful Commands

```bash
# Build
make build

# Tests
make test

# Format code
make fmt

# Clean
make clean

# Help
make help
```

## Next Steps

- Configure your services and ingress to use external-dns
- Adjust the domain filter to your needs
- Configure the synchronization interval
- Consult the [complete documentation](README.md)
