#!/usr/bin/env bash
set -euo pipefail

# Canonical spec: embedded into the backend and served at /api/docs/openapi.yaml.
SPEC_FILE="${1:-backend/handlers/openapi.yaml}"

echo "=========================================="
echo " Validating OpenAPI Specification: $SPEC_FILE"
echo "=========================================="

if [ ! -f "$SPEC_FILE" ]; then
    echo "❌ Error: OpenAPI specification file '$SPEC_FILE' not found!"
    exit 1
fi

# Run Go-based spec validation: structure, plus spec <-> Gin route parity.
echo "Running Go OpenAPI syntax, schema and route-parity verification..."
cd backend && go mod download && go test -v -run 'TestOpenAPISpecValidation|TestOpenAPISpecMatchesRoutes|TestSpecRoutesExpansion' .

echo ""
echo "✅ OpenAPI Specification is valid and passed all validation checks!"
