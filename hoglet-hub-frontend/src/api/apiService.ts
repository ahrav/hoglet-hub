import axios, { AxiosError, AxiosRequestConfig } from "axios";
import { API_BASE_URL } from "./config";
import { ApiError } from "./generated/core/ApiError";
import { ApiRequestOptions } from "./generated/core/ApiRequestOptions";
import { ApiResult } from "./generated/core/ApiResult";

// Token refresh state management
// TODO: Flesh this out some more
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason?: any) => void;
}> = [];

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach((request) => {
    if (error) {
      request.reject(error);
    } else {
      request.resolve(token);
    }
  });

  failedQueue = [];
};

const apiClient = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor for adding auth token
apiClient.interceptors.request.use(
  (config) => {
    if (typeof window !== "undefined") {
      const token = localStorage.getItem("auth_token");
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Helper to convert Axios error to the existing ApiError format
const createApiError = (error: AxiosError): ApiError => {
  // Ensure method is one of the valid HTTP methods
  const methodStr = error.config?.method?.toUpperCase() || "";
  const method = (
    ["GET", "DELETE", "HEAD", "OPTIONS", "POST", "PUT", "PATCH"].includes(
      methodStr
    )
      ? methodStr
      : "GET"
  ) as "GET" | "DELETE" | "HEAD" | "OPTIONS" | "POST" | "PUT" | "PATCH";

  const request: ApiRequestOptions = {
    method: method,
    url: error.config?.url || "unknown-url",
    path: {},
    cookies: {},
    headers: error.config?.headers || {},
    query: {},
    body: error.config?.data,
  };

  // Safely extract error message
  let errorMessage = "An error occurred";
  if (error.message) {
    errorMessage = error.message;
  }

  // Try to get a more specific message from the response if available
  const responseData = error.response?.data as Record<string, any> | undefined;
  if (
    responseData &&
    typeof responseData === "object" &&
    "message" in responseData
  ) {
    errorMessage = String(responseData.message);
  }

  const response: ApiResult = {
    url: error.config?.url || "unknown-url",
    ok: false,
    status: error.response?.status || 0,
    statusText: error.response?.statusText || "Unknown Error",
    body: error.response?.data,
  };

  return new ApiError(request, response, errorMessage);
};

// Response interceptor for handling common errors
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    // Original request configuration
    const originalRequest = error.config as AxiosRequestConfig & {
      _retry?: boolean;
    };

    if (error.response) {
      const { status } = error.response;

      // Handle authentication errors
      if (status === 401) {
        // Handle token refresh if this isn't already a retry
        if (
          originalRequest &&
          !originalRequest._retry &&
          typeof window !== "undefined"
        ) {
          if (isRefreshing) {
            // If already refreshing, queue this request
            return new Promise((resolve, reject) => {
              failedQueue.push({ resolve, reject });
            })
              .then((token) => {
                originalRequest.headers = {
                  ...originalRequest.headers,
                  Authorization: `Bearer ${token}`,
                };
                return apiClient(originalRequest);
              })
              .catch((err) => {
                return Promise.reject(err);
              });
          }

          originalRequest._retry = true;
          isRefreshing = true;

          try {
            // TODO: Implement token refresh
            // For now, just handle the 401 by redirecting to login
            processQueue(new Error("Failed to refresh token"), null);
            console.error("Authentication error:", error);
            window.location.href = "/login";
          } catch (refreshError) {
            processQueue(refreshError, null);
            localStorage.removeItem("auth_token");
            localStorage.removeItem("refresh_token");
            window.location.href = "/login";
            return Promise.reject(createApiError(error));
          } finally {
            isRefreshing = false;
          }
        } else {
          console.error("Authentication error:", error);
          // Simple redirect for unauthenticated requests
          // that don't need token refresh
          window.location.href = "/login";
        }
      } else if (status === 403) {
        console.error("Forbidden:", error);
      } else if (status === 404) {
        console.error("Not found:", error);
      } else if (status === 409) {
        console.error("Conflict:", error);
      } else if (status >= 500) {
        console.error("Server error:", error);
      }

      return Promise.reject(createApiError(error));
    }

    // Network errors, CORS issues, or server not responding
    if (error.request && !error.response) {
      console.error("Network error:", error);
      return Promise.reject(createApiError(error));
    }

    // Something else happened while setting up the request
    return Promise.reject(createApiError(error));
  }
);

export { apiClient };
