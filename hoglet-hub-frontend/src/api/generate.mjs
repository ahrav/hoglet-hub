#!/usr/bin/env node

import { generate } from "openapi-typescript-codegen";
import path from "path";
import { fileURLToPath } from "url";
import fs from "fs";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Path to the OpenAPI specification
const apiSpecPath = path.join(__dirname, "openapi.yaml");
const outputPath = path.join(__dirname, "generated");

// Validate the OpenAPI spec file exists
if (!fs.existsSync(apiSpecPath)) {
  console.error(`ERROR: OpenAPI specification not found at ${apiSpecPath}`);
  process.exit(1);
}

// Ensure output directory exists
if (!fs.existsSync(outputPath)) {
  console.log(`Creating output directory: ${outputPath}`);
  fs.mkdirSync(outputPath, { recursive: true });
}

console.log(`Generating API client from: ${apiSpecPath}`);
console.log(`Output directory: ${outputPath}`);

// Generate API client
async function generateApiClient() {
  try {
    await generate({
      input: apiSpecPath,
      output: outputPath,
      exportSchemas: true,
      exportServices: true,
      useOptions: true,
      useUnionTypes: true,
    });

    // Verify files were generated
    const files = fs.readdirSync(outputPath);
    if (files.length === 0) {
      throw new Error("No files were generated");
    }

    // Check if services directory was created
    const servicesPath = path.join(outputPath, "services");
    if (!fs.existsSync(servicesPath)) {
      throw new Error("Services directory was not generated");
    }

    console.log("API client generated successfully!");
    console.log("Generated files:", files);
  } catch (err) {
    console.error("Error generating API client:", err);
    process.exit(1);
  }
}

generateApiClient();
