"use client";

import { useCallback, useMemo } from "react";
import { OperationResponse } from "../api/generated/models/OperationResponse";

interface OperationDetailsProps {
  operation: OperationResponse;
  isLoading?: boolean;
}

export default function OperationDetails({
  operation,
  isLoading = false,
}: OperationDetailsProps) {
  const formatDate = useCallback((dateString: string) => {
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) {
        return "Invalid date";
      }
      return date.toLocaleString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
    } catch (e) {
      console.error("Date formatting error:", e);
      return "Invalid date";
    }
  }, []);

  const StatusIndicator = useMemo(() => {
    const statusClasses = {
      completed: "bg-green-100 text-green-800",
      failed: "bg-red-100 text-red-800",
      cancelled: "bg-yellow-100 text-yellow-800",
      in_progress: "bg-blue-100 text-blue-800",
      pending: "bg-blue-100 text-blue-800",
      default: "bg-gray-100 text-gray-800",
    };

    const statusClass =
      operation.status in statusClasses
        ? statusClasses[operation.status as keyof typeof statusClasses]
        : statusClasses.default;

    return (
      <span
        className={`px-3 py-1 rounded-full text-sm font-medium ${statusClass}`}
        role="status"
      >
        {operation.status}
      </span>
    );
  }, [operation.status]);

  const ParametersJSON = useMemo(() => {
    return operation.parameters && Object.keys(operation.parameters).length > 0
      ? JSON.stringify(operation.parameters, null, 2)
      : null;
  }, [operation.parameters]);

  const ResultJSON = useMemo(() => {
    return operation.result && Object.keys(operation.result).length > 0
      ? JSON.stringify(operation.result, null, 2)
      : null;
  }, [operation.result]);

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6 animate-pulse">
        <div className="h-8 bg-gray-200 rounded w-1/2 mb-4"></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          {[...Array(6)].map((_, i) => (
            <div key={i}>
              <div className="h-4 bg-gray-200 rounded w-24 mb-2"></div>
              <div className="h-6 bg-gray-200 rounded w-32"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <article className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-6">
      <header className="flex justify-between items-start mb-4">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
          Operation #{operation.id}
        </h2>
        {StatusIndicator}
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <div>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Operation Type
          </p>
          <p className="font-medium text-gray-900 dark:text-white">
            {operation.operation_type}
          </p>
        </div>

        {operation.tenant_id && (
          <div>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Tenant ID
            </p>
            <p className="font-medium text-gray-900 dark:text-white">
              {operation.tenant_id}
            </p>
          </div>
        )}

        <div>
          <p className="text-sm text-gray-500 dark:text-gray-400">Created By</p>
          <p className="font-medium text-gray-900 dark:text-white">
            {operation.created_by}
          </p>
        </div>

        <div>
          <p className="text-sm text-gray-500 dark:text-gray-400">Created At</p>
          <p className="font-medium text-gray-900 dark:text-white">
            {formatDate(operation.created_at)}
          </p>
        </div>

        {operation.started_at && (
          <div>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Started At
            </p>
            <p className="font-medium text-gray-900 dark:text-white">
              {formatDate(operation.started_at)}
            </p>
          </div>
        )}

        {operation.completed_at && (
          <div>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Completed At
            </p>
            <p className="font-medium text-gray-900 dark:text-white">
              {formatDate(operation.completed_at)}
            </p>
          </div>
        )}
      </div>

      {ParametersJSON && (
        <section className="mb-6">
          <h3 className="text-lg font-semibold mb-2 text-gray-900 dark:text-white">
            Parameters
          </h3>
          <pre className="bg-gray-50 dark:bg-gray-900 p-3 rounded-md text-sm overflow-x-auto">
            {ParametersJSON}
          </pre>
        </section>
      )}

      {ResultJSON && (
        <section className="mb-6">
          <h3 className="text-lg font-semibold mb-2 text-gray-900 dark:text-white">
            Result
          </h3>
          <pre className="bg-gray-50 dark:bg-gray-900 p-3 rounded-md text-sm overflow-x-auto">
            {ResultJSON}
          </pre>
        </section>
      )}

      {operation.error_message && (
        <section className="mb-6" aria-live="polite">
          <h3 className="text-lg font-semibold mb-2 text-red-600">Error</h3>
          <div className="bg-red-50 dark:bg-red-900/20 p-3 rounded-md text-red-700 dark:text-red-400">
            {operation.error_message}
          </div>
        </section>
      )}

      {operation._links && Object.keys(operation._links).length > 0 && (
        <section>
          <h3 className="text-lg font-semibold mb-2 text-gray-900 dark:text-white">
            Links
          </h3>
          <div className="flex flex-wrap gap-2">
            {Object.entries(operation._links).map(([key, url]) => (
              <a
                key={key}
                href={url}
                className="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded-md text-sm hover:bg-blue-200 dark:hover:bg-blue-900/50 transition-colors"
                target="_blank"
                rel="noopener noreferrer"
                aria-label={`Open ${key} link`}
              >
                {key}
              </a>
            ))}
          </div>
        </section>
      )}
    </article>
  );
}
