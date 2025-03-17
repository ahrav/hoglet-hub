import axios, { AxiosError } from "axios";
import { API_BASE_URL } from "./config";

// Create an axios instance for direct API calls
// For axios, we need to manually add the /api/v1 path
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

// Response interceptor for handling common errors
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // Handle authentication errors
    if (error.response?.status === 401) {
      // Redirect to login or refresh token
      console.error("Authentication error:", error);
      // You might want to redirect to login page or refresh the token
      // window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export { apiClient };
