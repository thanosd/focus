#!/bin/bash
set -e

echo "🔧 Generating API types from OpenAPI specification..."
echo ""

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

# Step 1: Bundle the OpenAPI spec (resolve all $ref to external files)
echo "📦 Step 1: Bundling OpenAPI spec..."
npx --yes @redocly/cli bundle openapi.yaml -o openapi-bundled.yaml
echo "✅ Bundle created: openapi-bundled.yaml"
echo ""

# Step 2: Generate TypeScript types for frontend
echo "📝 Step 2: Generating TypeScript types..."
cd ../frontend
npm run generate:api
echo "✅ TypeScript types generated: frontend/src/lib/api-types.ts"
echo ""

# Step 3: Generate Go types for backend. oapi-codegen can't read OpenAPI 3.1
# nullable type arrays, so it gets a 3.0-flavoured copy of the bundle.
echo "🔨 Step 3: Generating Go types..."
cd ../api
node to-oapi30.mjs openapi-bundled.yaml openapi-bundled-go.yaml
cd ../backend
OAPI_CODEGEN="${OAPI_CODEGEN:-$(command -v oapi-codegen || echo "${HOME}/go/bin/oapi-codegen")}"
"${OAPI_CODEGEN}" -config api-config.yaml ../api/openapi-bundled-go.yaml
rm -f ../api/openapi-bundled-go.yaml
echo "✅ Go types generated: backend/internal/api/types.go"
echo ""

echo "🎉 All API types generated successfully!"
