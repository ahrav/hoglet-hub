"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from "react";

interface AuthContextType {
  isAuthenticated: boolean;
  token: string | null;
  login: (token: string) => void;
  logout: () => void;
  error: string | null;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Safe localStorage helpers
const getLocalStorageItem = (key: string): string | null => {
  if (typeof window !== "undefined") {
    return localStorage.getItem(key);
  }
  return null;
};

const setLocalStorageItem = (key: string, value: string): void => {
  if (typeof window !== "undefined") {
    localStorage.setItem(key, value);
  }
};

const removeLocalStorageItem = (key: string): void => {
  if (typeof window !== "undefined") {
    localStorage.removeItem(key);
  }
};

export function AuthProvider({ children }: { children: ReactNode }) {
  const isDev = process.env.NEXT_PUBLIC_DEV_MODE === "true";

  const [token, setToken] = useState<string | null>(isDev ? "dev-token" : null);
  const [isAuthenticated, setIsAuthenticated] = useState(isDev);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isDev) {
      // TODO: Is this still acceptable, lol?
      const storedToken = getLocalStorageItem("auth_token");
      if (storedToken) {
        setToken(storedToken);
        setIsAuthenticated(true);
      }
    }
  }, [isDev]);

  const login = (newToken: string) => {
    try {
      setLocalStorageItem("auth_token", newToken);
      setToken(newToken);
      setIsAuthenticated(true);
      setError(null);
    } catch (err) {
      setError("Authentication failed");
      console.error(err);
    }
  };

  const logout = () => {
    if (isDev) {
      console.log("Logout clicked - disabled for development");
      return;
    }

    removeLocalStorageItem("auth_token");
    setToken(null);
    setIsAuthenticated(false);
  };

  const clearError = () => setError(null);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        token,
        login,
        logout,
        error,
        clearError,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
