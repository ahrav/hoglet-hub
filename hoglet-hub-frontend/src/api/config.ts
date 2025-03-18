import { OpenAPI } from "./generated";

export const API_BASE_URL =
  process.env.NODE_ENV === "development"
    ? "/api" // Use local proxy in development
    : process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const setupAPI = () => {
  // Set the base URL
  OpenAPI.BASE = API_BASE_URL;
  OpenAPI.WITH_CREDENTIALS = true;

  OpenAPI.TOKEN = () => {
    if (typeof window !== "undefined") {
      return Promise.resolve(localStorage.getItem("auth_token") || "");
    }
    return Promise.resolve("");
  };

  console.log(`API configured with base URL: ${API_BASE_URL}`);
};
