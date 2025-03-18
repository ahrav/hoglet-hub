import { NextRequest } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type ProxyParams = {
  proxy: string[];
};

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type, Authorization",
  "Access-Control-Max-Age": "86400",
};

const SECURITY_HEADERS = {
  "X-Content-Type-Options": "nosniff",
  "X-Frame-Options": "DENY",
  "X-XSS-Protection": "1; mode=block",
};

const FORWARDABLE_HEADERS = [
  "content-type",
  "authorization",
  "user-agent",
  "accept",
  "accept-language",
  "if-none-match",
  "if-modified-since",
  "cookie",
  "x-requested-with",
  "cache-control",
];

/**
 * This is a proxy handler for API requests.
 * It forwards requests to the backend API and adds CORS headers to the response.
 */
export async function GET(
  request: NextRequest,
  context: { params: ProxyParams }
): Promise<Response> {
  const params = context.params;
  return await proxyRequest(request, params.proxy, "GET");
}

export async function POST(
  request: NextRequest,
  context: { params: ProxyParams }
): Promise<Response> {
  const params = context.params;
  return await proxyRequest(request, params.proxy, "POST");
}

export async function PUT(
  request: NextRequest,
  context: { params: ProxyParams }
): Promise<Response> {
  const params = context.params;
  return await proxyRequest(request, params.proxy, "PUT");
}

export async function DELETE(
  request: NextRequest,
  context: { params: ProxyParams }
): Promise<Response> {
  const params = context.params;
  return await proxyRequest(request, params.proxy, "DELETE");
}

export async function OPTIONS(
  request: NextRequest,
  context: { params: ProxyParams }
): Promise<Response> {
  context.params;

  return new Response(null, {
    status: 204,
    headers: CORS_HEADERS,
  });
}

/**
 * Helper function to proxy requests to the backend API
 */
async function proxyRequest(
  request: NextRequest,
  pathSegments: string[],
  method: string
): Promise<Response> {
  const url = new URL(request.url);
  const targetUrl = `${API_URL}/${pathSegments.join("/")}${url.search}`;

  console.log(`Proxying ${method} request to: ${targetUrl}`, {
    path: pathSegments.join("/"),
    query: url.search,
    timestamp: new Date().toISOString(),
  });

  const headers: HeadersInit = {};

  // Only forward specific headers
  FORWARDABLE_HEADERS.forEach((headerName) => {
    const headerValue = request.headers.get(headerName);
    if (headerValue) {
      headers[headerName] = headerValue;
    }
  });

  try {
    let body: BodyInit | null | undefined = undefined;

    if (method !== "GET" && method !== "HEAD") {
      const contentType = request.headers.get("content-type") || "";

      if (contentType.includes("application/json")) {
        const json = await request.json();
        body = JSON.stringify(json);
      } else if (contentType.includes("multipart/form-data")) {
        body = await request.formData();
      } else if (contentType.includes("application/x-www-form-urlencoded")) {
        body = await request.text();
      } else {
        body = await request.arrayBuffer();
      }
    }

    // TODO: Implement intelligent caching strategy
    // For GET requests, consider caching responses except for frequently polled endpoints like operations
    // Example approach:
    // - For /api/v1/operations/* endpoints: no caching to support polling
    // - For other GET endpoints: cache with "s-maxage=60, stale-while-revalidate"
    // - Alternatively, implement cache-busting with timestamp query parameters

    const response = await fetch(targetUrl, {
      method,
      headers,
      body,
      // No caching for now to avoid stale data issues with polling
      cache: "no-store",
    });

    const responseHeaders = new Headers();

    response.headers.forEach((value, key) => {
      responseHeaders.set(key, value);
    });
    Object.entries(CORS_HEADERS).forEach(([key, value]) => {
      responseHeaders.set(key, value);
    });
    Object.entries(SECURITY_HEADERS).forEach(([key, value]) => {
      responseHeaders.set(key, value);
    });

    const responseData = await response.arrayBuffer();

    return new Response(responseData, {
      status: response.status,
      statusText: response.statusText,
      headers: responseHeaders,
    });
  } catch (error) {
    console.error("Proxy error:", error);

    let errorMessage = "Failed to proxy request";
    let status = 500;

    if (error instanceof TypeError && error.message.includes("fetch")) {
      errorMessage =
        "Cannot connect to backend API. Please check if the API server is running.";
      status = 503; // Service Unavailable
    }

    return new Response(
      JSON.stringify({
        error: errorMessage,
        timestamp: new Date().toISOString(),
        path: pathSegments.join("/"),
      }),
      {
        status,
        headers: {
          "Content-Type": "application/json",
          ...CORS_HEADERS,
          ...SECURITY_HEADERS,
        },
      }
    );
  }
}
