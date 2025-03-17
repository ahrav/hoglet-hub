#!/bin/bash

# Script to generate the API client from the OpenAPI specification

# Ensure the directory structure exists
mkdir -p src/api/generated

# Check if the OpenAPI spec exists
if [ ! -f src/api/openapi.yaml ]; then
  echo "Error: OpenAPI specification not found at src/api/openapi.yaml"
  echo "Make sure the OpenAPI spec is copied to the frontend using 'make update-frontend-api'"
  exit 1
fi

# Generate the API client
echo "Generating API client..."
npm run generate-api

# Check if generation was successful
if [ $? -ne 0 ]; then
  echo "Error: Failed to generate API client"
  exit 1
fi

# Check if the services directory exists
if [ ! -d src/api/generated/services ]; then
  echo "Error: Generated services directory not found"
  echo "API client generation failed or produced unexpected output"
  exit 1
fi

echo "API client generated successfully!"
echo "Generated files:"
ls -la src/api/generated/services

echo ""
echo "Next steps:"
echo "1. Run 'npm run build' to build the frontend"
echo "2. Run 'make docker-frontend' to build the Docker image"
echo "3. Run 'make dev-load' to load the image into your kind cluster"
echo ""

exit 0
