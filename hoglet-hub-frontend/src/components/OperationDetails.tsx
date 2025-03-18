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
      <div className="bg-gradient-to-b from-white to-blue-50 rounded-lg shadow-md p-6 animate-pulse border border-blue-100">
        <div className="h-8 bg-blue-200 rounded w-1/2 mb-4"></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          {[...Array(6)].map((_, i) => (
            <div key={i}>
              <div className="h-4 bg-blue-100 rounded w-24 mb-2"></div>
              <div className="h-6 bg-blue-100 rounded w-32"></div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <article className="bg-gradient-to-b from-white to-blue-50 rounded-lg shadow-md p-6 border-t-4 border-blue-500">
      <header className="flex justify-between items-start mb-6 pb-4 border-b border-blue-100">
        <h2 className="text-2xl font-bold text-blue-800">
          Operation #{operation.id}
        </h2>
        {StatusIndicator}
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="bg-white p-4 rounded-md shadow-sm border border-blue-100">
          <p className="text-sm font-medium text-blue-600 mb-1">
            Operation Type
          </p>
          <p className="font-semibold text-gray-900">
            {operation.operation_type}
          </p>
        </div>

        {operation.tenant_id && (
          <div className="bg-white p-4 rounded-md shadow-sm border border-blue-100">
            <p className="text-sm font-medium text-blue-600 mb-1">Tenant ID</p>
            <p className="font-semibold text-gray-900">{operation.tenant_id}</p>
          </div>
        )}

        <div className="bg-white p-4 rounded-md shadow-sm border border-blue-100">
          <p className="text-sm font-medium text-blue-600 mb-1">Created By</p>
          <p className="font-semibold text-gray-900">{operation.created_by}</p>
        </div>

        <div className="bg-white p-4 rounded-md shadow-sm border border-blue-100">
          <p className="text-sm font-medium text-blue-600 mb-1">Created At</p>
          <p className="font-semibold text-gray-900">
            {formatDate(operation.created_at)}
          </p>
        </div>

        {operation.started_at && (
          <div className="bg-white p-4 rounded-md shadow-sm border border-blue-100">
            <p className="text-sm font-medium text-blue-600 mb-1">Started At</p>
            <p className="font-semibold text-gray-900">
              {formatDate(operation.started_at)}
            </p>
          </div>
        )}

        {operation.completed_at && (
          <div className="bg-white p-4 rounded-md shadow-sm border border-blue-100">
            <p className="text-sm font-medium text-blue-600 mb-1">
              Completed At
            </p>
            <p className="font-semibold text-gray-900">
              {formatDate(operation.completed_at)}
            </p>
          </div>
        )}
      </div>

      {ParametersJSON && (
        <section className="mb-8">
          <h3 className="text-lg font-semibold mb-3 text-blue-800 pb-1 border-b border-blue-100">
            Parameters
          </h3>
          <pre className="bg-white p-4 rounded-md text-sm overflow-x-auto border border-blue-100 shadow-sm text-blue-900">
            {ParametersJSON}
          </pre>
        </section>
      )}

      {ResultJSON && (
        <section className="mb-8">
          <h3 className="text-lg font-semibold mb-3 text-blue-800 pb-1 border-b border-blue-100">
            Result
          </h3>
          <pre className="bg-white p-4 rounded-md text-sm overflow-x-auto border border-blue-100 shadow-sm text-blue-900">
            {ResultJSON}
          </pre>
        </section>
      )}

      {operation.error_message && (
        <section className="mb-8" aria-live="polite">
          <h3 className="text-lg font-semibold mb-3 text-red-600 pb-1 border-b border-red-100">
            Error
          </h3>
          <div className="bg-red-50 p-4 rounded-md text-red-800 border border-red-200 shadow-sm">
            {operation.error_message}
          </div>
        </section>
      )}

      {operation._links && Object.keys(operation._links).length > 0 && (
        <section>
          <h3 className="text-lg font-semibold mb-3 text-blue-800 pb-1 border-b border-blue-100">
            Links
          </h3>
          <div className="flex flex-wrap gap-3">
            {Object.entries(operation._links).map(([key, url]) => (
              <a
                key={key}
                href={url}
                className="px-4 py-2 bg-blue-100 text-blue-800 rounded-md text-sm hover:bg-blue-200 transition-colors border border-blue-200 font-medium"
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
