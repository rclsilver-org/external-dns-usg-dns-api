#!/bin/bash

# Local test script for external-dns-usg-dns-api

set -e

echo "=== Testing external-dns-usg-dns-api webhook ==="
echo ""

# Check that environment variables are set
if [ -z "$USG_DNS_URL" ]; then
    echo "❌ Error: USG_DNS_URL is not set"
    echo "   Example: export USG_DNS_URL='http://192.168.1.1:8080'"
    exit 1
fi

if [ -z "$USG_DNS_TOKEN" ]; then
    echo "❌ Error: USG_DNS_TOKEN is not set"
    echo "   Example: export USG_DNS_TOKEN='your-token'"
    exit 1
fi

echo "✅ Configuration:"
echo "   USG_DNS_URL: $USG_DNS_URL"
echo "   DOMAIN_FILTER: ${DOMAIN_FILTER:-<no filter>}"
echo "   SERVER_PORT: ${SERVER_PORT:-8888}"
echo "   DRY_RUN: ${DRY_RUN:-false}"
echo ""

# Build the project
echo "📦 Building the project..."
go build -o external-dns-usg-dns-api ./cmd/external-dns-usg-dns-api
echo "✅ Build successful"
echo ""

# Test API connectivity
echo "🔍 Testing USG DNS API connectivity..."
if curl -sf -H "Authorization: $USG_DNS_TOKEN" "$USG_DNS_URL/records" > /dev/null; then
    echo "✅ API connection successful"
else
    echo "❌ Cannot connect to API"
    echo "   Check the URL and token"
    exit 1
fi
echo ""

# Start the webhook
echo "🚀 Starting the webhook..."
echo "   Webhook will be accessible at http://localhost:${SERVER_PORT:-8888}"
echo "   Health check: http://localhost:8080/healthz"
echo ""
echo "   To test manually:"
echo "   curl http://localhost:${SERVER_PORT:-8888}/"
echo "   curl http://localhost:${SERVER_PORT:-8888}/records"
echo ""
echo "   Press Ctrl+C to stop"
echo ""

./external-dns-usg-dns-api
