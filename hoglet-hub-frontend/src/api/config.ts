import { OpenAPI } from "./generated";

// Set the API base URL globally
const API_BASE_URL =
  process.env.NODE_ENV === "development"
    ? "/api" // Use our local proxy
    : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// Initialize the OpenAPI configuration (only once at startup)
const initializeApi = () => {
  // Set the base URL (without the /api/v1 path to avoid duplication)
  OpenAPI.BASE = API_BASE_URL;
  OpenAPI.WITH_CREDENTIALS = true;

  // Configure authentication
  OpenAPI.TOKEN = () => {
    if (typeof window !== "undefined") {
      return Promise.resolve(localStorage.getItem("auth_token") || "");
    }
    return Promise.resolve("");
  };

  console.log(`API configured with base URL: ${API_BASE_URL}`);
};

// Run the initialization immediately
initializeApi();

export { API_BASE_URL, initializeApi };
