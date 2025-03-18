"use client";

import { useCallback, useState, useEffect } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  tenantCreateSchema,
  TenantCreateFormData,
} from "../schemas/tenantSchema";
import { useTenantApi } from "../hooks/useTenantApi";
import { OperationStatus } from "../api/generated/models/OperationStatus";

export default function TenantCreateForm() {
  const { createTenant, getOperation } = useTenantApi();
  const [operationId, setOperationId] = useState<number | null>(null);
  const [formState, setFormState] = useState<
    "idle" | "submitting" | "success" | "error" | "pending"
  >("idle");

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

  const operationQuery = getOperation(operationId);
  useEffect(() => {
    if (!operationQuery.data) return;

    if (operationQuery.data.isComplete) {
      if (operationQuery.data.status === "completed") {
        setFormState("success");
      } else {
        setFormState("error");
      }
    } else if (operationQuery.data.status === "pending") {
      setFormState("pending");
    }
  }, [operationQuery.data]);

  const getStatusColor = useCallback((status?: OperationStatus): string => {
    if (!status) return "text-gray-500"; // Default color for undefined

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
      setFormState("submitting");

      try {
        const result = await createTenant.mutateAsync(data);
        if (result && result.operation_id) {
          setOperationId(result.operation_id);
        }
      } catch (error) {
        console.error("Error creating tenant:", error);
        setFormState("error");
      }
    },
    [createTenant]
  );

  const handleReset = useCallback(() => {
    resetForm();
    setOperationId(null);
    setFormState("idle");
  }, [resetForm]);

  const renderOperationStatus = useCallback(() => {
    if (!operationQuery.data) return null;

    return (
      <div
        className="mt-4 p-4 rounded-md border"
        role="status"
        aria-live="polite"
      >
        <h3 className="text-lg font-medium">Operation Status</h3>
        <p className="mt-1">
          <span className="font-medium">Status:</span>{" "}
          <span className={`${getStatusColor(operationQuery.data.status)}`}>
            {operationQuery.data.status}
          </span>
        </p>
        {operationQuery.data.error_message && (
          <p className="mt-1 text-red-500">
            Error: {operationQuery.data.error_message}
          </p>
        )}
        {operationQuery.data.status === "completed" && (
          <p className="mt-1 text-green-500">Tenant successfully created!</p>
        )}
        {operationQuery.data.status === "pending" && (
          <p className="mt-1 text-blue-500">
            Tenant is being created. This may take a few minutes.
          </p>
        )}
      </div>
    );
  }, [operationQuery.data, getStatusColor]);

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
                disabled={formState !== "idle"}
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
                disabled={formState !== "idle"}
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
                disabled={formState !== "idle"}
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
            disabled={formState !== "idle" || createTenant.isPending}
            aria-busy={formState === "submitting" || createTenant.isPending}
            className={`w-full py-2 px-4 rounded-md text-white font-medium ${
              formState === "idle" && !createTenant.isPending
                ? "bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                : "bg-gray-400 cursor-not-allowed"
            }`}
          >
            {formState === "idle" && !createTenant.isPending && "Create Tenant"}
            {(formState === "submitting" || createTenant.isPending) &&
              "Creating..."}
            {formState === "pending" && "Provisioning..."}
            {formState === "success" && "Created Successfully"}
            {formState === "error" && "Error"}
          </button>
        </div>
      </form>

      {createTenant.isError && (
        <div
          className="mt-4 p-3 bg-red-100 text-red-700 rounded-md"
          role="alert"
        >
          <p>
            Error:{" "}
            {createTenant.error instanceof Error
              ? createTenant.error.message
              : "Failed to create tenant"}
          </p>
        </div>
      )}

      {operationQuery.isError && (
        <div
          className="mt-4 p-3 bg-red-100 text-red-700 rounded-md"
          role="alert"
        >
          <p>
            Error:{" "}
            {operationQuery.error instanceof Error
              ? operationQuery.error.message
              : "Failed to fetch operation status"}
          </p>
        </div>
      )}

      {operationId && renderOperationStatus()}

      {(formState === "success" || formState === "error") && (
        <div className="mt-4">
          <button
            onClick={handleReset}
            className="w-full py-2 px-4 bg-gray-200 hover:bg-gray-300 rounded-md text-gray-800 font-medium"
            type="button"
          >
            {formState === "success" ? "Create Another Tenant" : "Try Again"}
          </button>
        </div>
      )}
    </div>
  );
}
