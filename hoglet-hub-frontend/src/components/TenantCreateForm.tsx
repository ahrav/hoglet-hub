"use client";

import { useCallback, useMemo, useReducer } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  tenantCreateSchema,
  TenantCreateFormData,
} from "../schemas/tenantSchema";
import { useTenantApi } from "../hooks/useTenantApi";
import { OperationStatus } from "../api/generated/models/OperationStatus";
import { OperationResponse } from "../api/generated/models/OperationResponse";

type FormState = {
  status: "idle" | "submitting" | "polling" | "success" | "error" | "pending";
  operationResult: OperationResponse | null;
  error: Error | null;
};

type FormAction =
  | { type: "SUBMIT" }
  | { type: "POLLING" }
  | { type: "SUCCESS"; result: OperationResponse }
  | { type: "ERROR"; error: Error }
  | { type: "PENDING"; result: OperationResponse }
  | { type: "RESET" };

function formReducer(state: FormState, action: FormAction): FormState {
  switch (action.type) {
    case "SUBMIT":
      return { ...state, status: "submitting", error: null };
    case "POLLING":
      return { ...state, status: "polling" };
    case "SUCCESS":
      return {
        ...state,
        status: "success",
        operationResult: action.result,
      };
    case "ERROR":
      return {
        ...state,
        status: "error",
        error: action.error,
      };
    case "PENDING":
      return {
        ...state,
        status: "pending",
        operationResult: action.result,
      };
    case "RESET":
      return {
        status: "idle",
        operationResult: null,
        error: null,
      };
  }
}

export default function TenantCreateForm() {
  const { error: apiError, createTenant, pollOperationStatus } = useTenantApi();

  const [state, dispatch] = useReducer(formReducer, {
    status: "idle",
    operationResult: null,
    error: null,
  });

  const {
    control,
    handleSubmit,
    formState: { errors },
    reset: resetForm,
  } = useForm<TenantCreateFormData>({
    resolver: zodResolver(tenantCreateSchema),
    defaultValues: {
      name: "",
      region: "us1",
      tier: "free",
      isolation_group_id: null,
    },
  });

  const getStatusColor = useCallback((status: OperationStatus): string => {
    switch (status) {
      case "completed":
        return "text-green-500";
      case "failed":
        return "text-red-500";
      case "cancelled":
        return "text-yellow-500";
      case "pending":
        return "text-blue-500";
      default:
        return "text-blue-500";
    }
  }, []);

  const onSubmit = useCallback(
    async (data: TenantCreateFormData) => {
      dispatch({ type: "SUBMIT" });

      // TODO: Come back and refactor this monstrosity.
      try {
        const response = await createTenant(data);
        if (response) {
          dispatch({ type: "POLLING" });

          try {
            const operationResult = await pollOperationStatus(
              response.operation_id
            );

            if (operationResult) {
              if (operationResult.status === "completed") {
                dispatch({ type: "SUCCESS", result: operationResult });
              } else if (
                operationResult.status === "failed" ||
                operationResult.status === "cancelled"
              ) {
                dispatch({
                  type: "ERROR",
                  error: new Error(
                    operationResult.error_message ||
                      `Operation ${operationResult.status}`
                  ),
                });
              } else {
                dispatch({ type: "PENDING", result: operationResult });
              }
            } else {
              dispatch({
                type: "ERROR",
                error: new Error("Failed to retrieve operation result"),
              });
            }
          } catch (pollError) {
            console.error("Error polling operation status:", pollError);
            dispatch({
              type: "ERROR",
              error:
                pollError instanceof Error
                  ? pollError
                  : new Error("Failed to poll operation status"),
            });
          }
        } else {
          dispatch({
            type: "ERROR",
            error: new Error("No response received from create tenant API"),
          });
        }
      } catch (err) {
        console.error("Error creating tenant:", err);
        dispatch({
          type: "ERROR",
          error:
            err instanceof Error ? err : new Error("Unknown error occurred"),
        });
      }
    },
    [createTenant, pollOperationStatus]
  );

  const handleReset = useCallback(() => {
    resetForm();
    dispatch({ type: "RESET" });
  }, [resetForm]);

  const renderOperationStatus = useCallback(() => {
    if (!state.operationResult) return null;

    return (
      <div
        className="mt-4 p-4 rounded-md border"
        role="status"
        aria-live="polite"
      >
        <h3 className="text-lg font-medium">Operation Status</h3>
        <p className="mt-1">
          <span className="font-medium">Status:</span>{" "}
          <span className={`${getStatusColor(state.operationResult.status)}`}>
            {state.operationResult.status}
          </span>
        </p>
        {state.operationResult.error_message && (
          <p className="mt-1 text-red-500">
            Error: {state.operationResult.error_message}
          </p>
        )}
        {state.operationResult.status === "completed" && (
          <p className="mt-1 text-green-500">Tenant successfully created!</p>
        )}
        {state.operationResult.status === "pending" && (
          <p className="mt-1 text-blue-500">
            Tenant is being created. This may take a few minutes.
          </p>
        )}
      </div>
    );
  }, [state.operationResult, getStatusColor]);

  return (
    <div className="w-full max-w-md mx-auto p-6 bg-white dark:bg-gray-800 rounded-lg shadow-md">
      <h2 className="text-2xl font-bold mb-6 text-gray-800 dark:text-white">
        Create New Tenant
      </h2>

      <form
        onSubmit={handleSubmit(onSubmit)}
        className="space-y-4"
        aria-label="Create tenant form"
      >
        <div>
          <label
            id="name-label"
            htmlFor="name"
            className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1"
          >
            Tenant Name
          </label>
          <Controller
            name="name"
            control={control}
            render={({ field }) => (
              <input
                {...field}
                type="text"
                id="name"
                aria-labelledby="name-label"
                aria-invalid={!!errors.name}
                aria-describedby={errors.name ? "name-error" : undefined}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                placeholder="tenant-name"
                disabled={state.status !== "idle"}
              />
            )}
          />
          {errors.name && (
            <p
              id="name-error"
              className="mt-1 text-sm text-red-600"
              role="alert"
            >
              {errors.name.message}
            </p>
          )}
        </div>

        <div>
          <label
            id="region-label"
            htmlFor="region"
            className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1"
          >
            Region
          </label>
          <Controller
            name="region"
            control={control}
            render={({ field }) => (
              <select
                {...field}
                id="region"
                aria-labelledby="region-label"
                aria-invalid={!!errors.region}
                aria-describedby={errors.region ? "region-error" : undefined}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                disabled={state.status !== "idle"}
              >
                <option value="us1">US East (us1)</option>
                <option value="us2">US West (us2)</option>
                <option value="us3">US Central (us3)</option>
                <option value="us4">US South (us4)</option>
                <option value="eu1">Europe West (eu1)</option>
                <option value="eu2">Europe Central (eu2)</option>
                <option value="eu3">Europe North (eu3)</option>
                <option value="eu4">Europe South (eu4)</option>
              </select>
            )}
          />
          {errors.region && (
            <p
              id="region-error"
              className="mt-1 text-sm text-red-600"
              role="alert"
            >
              {errors.region.message}
            </p>
          )}
        </div>

        <div>
          <label
            id="tier-label"
            htmlFor="tier"
            className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1"
          >
            Tier
          </label>
          <Controller
            name="tier"
            control={control}
            render={({ field }) => (
              <select
                {...field}
                id="tier"
                aria-labelledby="tier-label"
                aria-invalid={!!errors.tier}
                aria-describedby={errors.tier ? "tier-error" : undefined}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white"
                disabled={state.status !== "idle"}
              >
                <option value="free">Free</option>
                <option value="pro">Pro</option>
                <option value="enterprise">Enterprise</option>
              </select>
            )}
          />
          {errors.tier && (
            <p
              id="tier-error"
              className="mt-1 text-sm text-red-600"
              role="alert"
            >
              {errors.tier.message}
            </p>
          )}
        </div>

        <div>
          <button
            type="submit"
            disabled={state.status !== "idle"}
            aria-busy={
              state.status === "submitting" || state.status === "polling"
            }
            className={`w-full py-2 px-4 rounded-md text-white font-medium ${
              state.status === "idle"
                ? "bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                : "bg-gray-400 cursor-not-allowed"
            }`}
          >
            {state.status === "idle" && "Create Tenant"}
            {state.status === "submitting" && "Creating..."}
            {state.status === "polling" && "Processing..."}
            {state.status === "pending" && "Provisioning..."}
            {state.status === "success" && "Created Successfully"}
            {state.status === "error" && "Error"}
          </button>
        </div>
      </form>

      {apiError && (
        <div
          className="mt-4 p-3 bg-red-100 text-red-700 rounded-md"
          role="alert"
        >
          <p>{apiError.message}</p>
        </div>
      )}

      {state.error && (
        <div
          className="mt-4 p-3 bg-red-100 text-red-700 rounded-md"
          role="alert"
        >
          <p>Error: {state.error.message}</p>
        </div>
      )}

      {state.status !== "idle" && renderOperationStatus()}

      {(state.status === "success" ||
        state.status === "pending" ||
        state.status === "error") && (
        <div className="mt-4">
          <button
            onClick={handleReset}
            className="w-full py-2 px-4 bg-gray-200 hover:bg-gray-300 rounded-md text-gray-800 font-medium"
            type="button"
          >
            {state.status === "success"
              ? "Create Another Tenant"
              : state.status === "error"
              ? "Try Again"
              : "Create New Tenant"}
          </button>
        </div>
      )}
    </div>
  );
}
