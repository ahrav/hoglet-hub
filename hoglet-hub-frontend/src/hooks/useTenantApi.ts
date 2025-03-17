"use client";

import { useState } from "react";
import { TenantCreate } from "../api/generated/models/TenantCreate";
import { TenantService } from "../api/services";
import { useAuth } from "../contexts/AuthContext";
import { AsyncOperation } from "../api/generated/models/AsyncOperation";
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

export function useTenantApi() {
  const { isAuthenticated } = useAuth();

  // Operation-specific loading states
  const [createLoading, setCreateLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [operationLoading, setOperationLoading] = useState(false);
  const [error, setError] = useState<ApiErrorState | null>(null);
  const [operationId, setOperationId] = useState<number | null>(null);
  const [operationDetails, setOperationDetails] =
    useState<OperationResponse | null>(null);

  const resetState = () => {
    setCreateLoading(false);
    setDeleteLoading(false);
    setOperationLoading(false);
    setError(null);
    setOperationId(null);
    setOperationDetails(null);
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

  // Create a new tenant
  const createTenant = async (
    tenantData: TenantCreate,
    retryOptions?: RetryOptions
  ): Promise<AsyncOperation | null> => {
    if (!isAuthenticated) {
      setError({
        operation: "createTenant",
        message: "You must be authenticated to perform this action",
      });
      return null;
    }

    setCreateLoading(true);
    setError(null);

    try {
      const response = await retryApiCall(
        () =>
          TenantService.createTenant({
            requestBody: tenantData,
          }),
        retryOptions
      );

      setOperationId(response.operation_id);
      return response;
    } catch (err: unknown) {
      let errorMessage = "Failed to create tenant";
      let statusCode;

      if (err instanceof ApiError) {
        errorMessage = err.message || errorMessage;
        statusCode = err.status;
      } else if (err instanceof Error) {
        errorMessage = err.message;
      }

      setError({
        operation: "createTenant",
        message: errorMessage,
        code: statusCode,
        details: err,
      });
      return null;
    } finally {
      setCreateLoading(false);
    }
  };

  // Delete an existing tenant
  const deleteTenant = async (
    tenantId: number,
    retryOptions?: RetryOptions
  ): Promise<AsyncOperation | null> => {
    if (!isAuthenticated) {
      setError({
        operation: "deleteTenant",
        message: "You must be authenticated to perform this action",
      });
      return null;
    }

    setDeleteLoading(true);
    setError(null);

    try {
      const response = await retryApiCall(
        () =>
          TenantService.deleteTenant({
            tenantId,
          }),
        retryOptions
      );

      setOperationId(response.operation_id);
      return response;
    } catch (err: unknown) {
      let errorMessage = "Failed to delete tenant";
      let statusCode;

      if (err instanceof ApiError) {
        errorMessage = err.message || errorMessage;
        statusCode = err.status;
      } else if (err instanceof Error) {
        errorMessage = err.message;
      }

      setError({
        operation: "deleteTenant",
        message: errorMessage,
        code: statusCode,
        details: err,
      });
      return null;
    } finally {
      setDeleteLoading(false);
    }
  };

  // Get operation details
  const getOperationDetails = async (
    opId: number,
    retryOptions?: RetryOptions
  ): Promise<OperationResponse | null> => {
    if (!isAuthenticated) {
      setError({
        operation: "getOperationDetails",
        message: "You must be authenticated to perform this action",
      });
      return null;
    }

    setOperationLoading(true);
    setError(null);

    try {
      const response = await retryApiCall(
        () =>
          OperationService.getOperation({
            operationId: opId,
          }),
        retryOptions
      );

      setOperationDetails(response);
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
        operation: "getOperationDetails",
        message: errorMessage,
        code: statusCode,
        details: err,
      });
      return null;
    } finally {
      setOperationLoading(false);
    }
  };

  // Poll operation status
  const pollOperationStatus = async (
    opId: number,
    interval = 2000,
    maxAttempts = 30
  ): Promise<OperationResponse | null> => {
    for (let attempts = 0; attempts < maxAttempts; attempts++) {
      const details = await getOperationDetails(opId);

      if (!details) return null;

      if (["completed", "failed", "cancelled"].includes(details.status)) {
        return details;
      }

      if (attempts < maxAttempts - 1) {
        await new Promise((resolve) => setTimeout(resolve, interval));
      }
    }

    setError({
      operation: "pollOperationStatus",
      message: "Operation polling timed out",
    });
    return null;
  };

  return {
    // Loading states
    createLoading,
    deleteLoading,
    operationLoading,
    isLoading: createLoading || deleteLoading || operationLoading,

    // States
    error,
    operationId,
    operationDetails,

    // Methods
    createTenant,
    deleteTenant,
    getOperationDetails,
    pollOperationStatus,
    resetState,
  };
}
