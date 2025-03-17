"use client";

import { useState } from "react";
import { useAuth } from "../contexts/AuthContext";
import { OperationResponse } from "../api/generated/models/OperationResponse";
import { ApiError } from "../api/generated/core/ApiError";
import { OperationService } from "../api/services";

interface ApiErrorState {
  operation: string;
  message: string;
  code?: number;
  details?: unknown;
}

interface RetryOptions {
  maxRetries?: number;
  initialDelay?: number;
  shouldRetry?: (error: unknown) => boolean;
}

export function useOperationsApi() {
  const { isAuthenticated } = useAuth();
  const [getOperationLoading, setGetOperationLoading] = useState(false);
  const [pollOperationLoading, setPollOperationLoading] = useState(false);
  const [error, setError] = useState<ApiErrorState | null>(null);
  const [operation, setOperation] = useState<OperationResponse | null>(null);

  // Reset hook state
  const resetState = () => {
    setGetOperationLoading(false);
    setPollOperationLoading(false);
    setError(null);
    setOperation(null);
  };

  async function retryApiCall<T>(
    apiCall: () => Promise<T>,
    options: RetryOptions = {}
  ): Promise<T> {
    const {
      maxRetries = 3,
      initialDelay = 1000,
      shouldRetry = (error: unknown) => {
        if (error instanceof ApiError) {
          // Don't retry client errors (except for 429 too many requests)
          if (
            [400, 401, 403, 404].includes(error.status) &&
            error.status !== 429
          ) {
            return false;
          }
        }
        return true;
      },
    } = options;

    let retries = 0;

    while (true) {
      try {
        return await apiCall();
      } catch (error) {
        if (!shouldRetry(error) || retries >= maxRetries) {
          throw error;
        }

        retries++;
        const backoffDelay = initialDelay * Math.pow(2, retries - 1);
        await new Promise((resolve) => setTimeout(resolve, backoffDelay));
      }
    }
  }

  // Get operation details
  const getOperation = async (
    operationId: number,
    retryOptions?: RetryOptions
  ): Promise<OperationResponse | null> => {
    if (!isAuthenticated) {
      setError({
        operation: "getOperation",
        message: "You must be authenticated to perform this action",
      });
      return null;
    }

    setGetOperationLoading(true);
    setError(null);

    try {
      const response = await retryApiCall(
        () => OperationService.getOperation({ operationId }),
        retryOptions
      );

      setOperation(response);
      return response;
    } catch (err: unknown) {
      let errorMessage = "Failed to fetch operation details";
      let statusCode;

      if (err instanceof ApiError) {
        errorMessage = err.message || errorMessage;
        statusCode = err.status;
      } else if (err instanceof Error) {
        errorMessage = err.message;
      }

      setError({
        operation: "getOperation",
        message: errorMessage,
        code: statusCode,
        details: err,
      });
      return null;
    } finally {
      setGetOperationLoading(false);
    }
  };

  // Poll operation status
  const pollOperationStatus = async (
    opId: number,
    interval = 2000,
    maxAttempts = 30
  ): Promise<OperationResponse | null> => {
    setPollOperationLoading(true);
    setError(null);

    try {
      for (let attempts = 0; attempts < maxAttempts; attempts++) {
        const details = await getOperation(opId);

        if (!details) return null;

        if (["completed", "failed", "cancelled"].includes(details.status)) {
          return details;
        }

        // Wait for the interval before the next attempt
        if (attempts < maxAttempts - 1) {
          await new Promise((resolve) => setTimeout(resolve, interval));
        }
      }

      setError({
        operation: "pollOperationStatus",
        message: "Operation polling timed out",
      });
      return null;
    } finally {
      setPollOperationLoading(false);
    }
  };

  return {
    // Loading states
    getOperationLoading,
    pollOperationLoading,
    isLoading: getOperationLoading || pollOperationLoading,

    // States
    error,
    operation,

    // Methods
    getOperation,
    pollOperationStatus,
    resetState,
  };
}
