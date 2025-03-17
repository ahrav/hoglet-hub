import { NextRequest } from "next/server";

// Backend API URL
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/**
 * This is a proxy handler for API requests.
 * It forwards requests to the backend API and adds CORS headers to the response.
 */
export async function GET(
  request: NextRequest,
  context: { params: { proxy: string[] } }
) {
  const params = await context.params;
  return await proxyRequest(request, params.proxy, "GET");
}

export async function POST(
  request: NextRequest,
  context: { params: { proxy: string[] } }
) {
  const params = await context.params;
  return await proxyRequest(request, params.proxy, "POST");
}

export async function PUT(
  request: NextRequest,
  context: { params: { proxy: string[] } }
) {
  const params = await context.params;
  return await proxyRequest(request, params.proxy, "PUT");
}

export async function DELETE(
  request: NextRequest,
  context: { params: { proxy: string[] } }
) {
  const params = await context.params;
  return await proxyRequest(request, params.proxy, "DELETE");
}

export async function OPTIONS(
  request: NextRequest,
  context: { params: { proxy: string[] } }
) {
  // We await params even though we don't use it, to follow Next.js best practices
  await context.params;

  // Handle CORS preflight request
  return new Response(null, {
    status: 204,
    headers: {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type, Authorization",
      "Access-Control-Max-Age": "86400",
    },
  });
}

/**
 * Helper function to proxy requests to the backend API
 */
async function proxyRequest(
  request: NextRequest,
  pathSegments: string[],
  method: string
) {
  const url = new URL(request.url);
  const targetUrl = `${API_URL}/${pathSegments.join("/")}${url.search}`;

  console.log(`Proxying ${method} request to: ${targetUrl}`);

  const headers: HeadersInit = {};
  request.headers.forEach((value, key) => {
    // Forward all headers except host
    if (key !== "host") {
      headers[key] = value;
    }
  });

  try {
    const response = await fetch(targetUrl, {
      method,
      headers,
      body:
        method !== "GET" && method !== "HEAD"
          ? await request.text()
          : undefined,
    });

    const responseHeaders = new Headers();
    response.headers.forEach((value, key) => {
      responseHeaders.set(key, value);
    });

    // Add CORS headers
    responseHeaders.set("Access-Control-Allow-Origin", "*");

    const responseData = await response.arrayBuffer();

    return new Response(responseData, {
      status: response.status,
      statusText: response.statusText,
      headers: responseHeaders,
    });
  } catch (error) {
    console.error("Proxy error:", error);
    return new Response(JSON.stringify({ error: "Failed to proxy request" }), {
      status: 500,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
    });
  }
}
