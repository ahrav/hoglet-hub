"use client"; // Still needed in Next.js App Router for any client components

import React, { useState, useEffect } from "react";
import {
  QueryClient,
  QueryClientProvider,
  QueryCache,
  MutationCache,
} from "@tanstack/react-query";
import { ErrorBoundary } from "react-error-boundary";
import dynamic from "next/dynamic";

// Dynamically import devtools to exclude from production bundles
const ReactQueryDevtools =
  process.env.NEXT_PUBLIC_DEV_MODE === "true"
    ? dynamic(() =>
        import("@tanstack/react-query-devtools").then(
          (mod) => mod.ReactQueryDevtools
        )
      )
    : () => null;

// Error fallback component
const QueryErrorFallback = ({
  error,
  resetErrorBoundary,
}: {
  error: Error;
  resetErrorBoundary: () => void;
}) => (
  <div role="alert" className="p-4 bg-red-50 border border-red-200 rounded-md">
    <h2 className="text-lg font-semibold text-red-800">Something went wrong</h2>
    <p className="text-sm text-red-600 mt-1">{error.message}</p>
    <button
      onClick={resetErrorBoundary}
      className="mt-3 px-3 py-1 bg-red-100 text-red-800 rounded hover:bg-red-200"
    >
      Try again
    </button>
  </div>
);

function makeQueryClient() {
  console.log("Creating new QueryClient instance");
  return new QueryClient({
    queryCache: new QueryCache({
      onError: (error) => {
        console.error("Query cache error:", error);
      },
    }),
    mutationCache: new MutationCache({
      onError: (error) => {
        console.error("Mutation cache error:", error);
      },
    }),
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: 1,
        staleTime: 60 * 1000, // 1 minute
        gcTime: 5 * 60 * 1000, // 5 minutes
      },
    },
  });
}

// This ensures the QueryClient is always created on the client
// Tbh, I don't really understand this stuff... but alas it works.
let browserQueryClient: QueryClient | undefined = undefined;

const getQueryClient = () => {
  if (typeof window === "undefined") {
    // Server: always return a new client
    return makeQueryClient();
  }

  // Browser: use singleton pattern
  if (!browserQueryClient) {
    browserQueryClient = makeQueryClient();
  }

  return browserQueryClient;
};

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(getQueryClient);

  useEffect(() => {
    console.log("QueryProvider mounted");
    return () => {
      console.log("QueryProvider unmounted");
    };
  }, []);

  return (
    <ErrorBoundary
      FallbackComponent={QueryErrorFallback}
      onReset={() => {
        console.log("Error boundary reset - clearing query client");
        queryClient.clear();
      }}
    >
      <QueryClientProvider client={queryClient}>
        {children}
        {process.env.NEXT_PUBLIC_DEV_MODE === "true" && <ReactQueryDevtools />}
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
