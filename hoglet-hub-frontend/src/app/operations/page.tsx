"use client";

import React, { useState, useEffect } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useOperationsApi } from "../../hooks/useOperationsApi";
import OperationDetails from "../../components/OperationDetails";

const PAGE_TITLE = "View Operations";
const LOOKUP_SECTION_TITLE = "Lookup an Operation";
const EMPTY_STATE_MESSAGE = "Enter an operation ID to view its details";

export default function OperationsPage(): React.ReactElement {
  const { isAuthenticated } = useAuth();
  const router = useRouter();
  const { getOperation, operation, isLoading, error } = useOperationsApi();
  const [operationId, setOperationId] = useState<string>("");
  const [isCheckingAuth, setIsCheckingAuth] = useState<boolean>(true);

  useEffect(() => {
    if (isAuthenticated === false) {
      router.push("/login");
    } else if (isAuthenticated === true) {
      setIsCheckingAuth(false);
    }
  }, [isAuthenticated, router]);

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    const id = parseInt(operationId.trim());
    if (!isNaN(id)) {
      await getOperation(id);
    }
  };

  if (isCheckingAuth) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <div className="text-xl text-gray-600">Loading...</div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto" role="main">
      <h1 className="text-3xl font-bold mb-8 text-center">{PAGE_TITLE}</h1>

      <div className="bg-white rounded-lg shadow-md p-6 mb-8">
        <h2 className="text-xl font-semibold mb-4">{LOOKUP_SECTION_TITLE}</h2>
        <form
          onSubmit={handleSubmit}
          className="flex gap-4"
          aria-label="Operation lookup form"
        >
          <input
            type="text"
            value={operationId}
            onChange={(e) => setOperationId(e.target.value)}
            placeholder="Enter Operation ID"
            className="flex-grow px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label="Operation ID"
          />
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-gray-400"
            aria-busy={isLoading}
          >
            {isLoading ? "Loading..." : "View"}
          </button>
        </form>

        {error && (
          <div
            className="mt-4 p-3 bg-red-100 text-red-700 rounded-md"
            role="alert"
          >
            <p>{error.message}</p>
          </div>
        )}
      </div>

      {operation && <OperationDetails operation={operation} />}

      {!operation && !isLoading && !error && (
        <div
          className="bg-gray-50 rounded-lg p-8 text-center"
          aria-live="polite"
        >
          <p className="text-gray-500 text-lg">{EMPTY_STATE_MESSAGE}</p>
        </div>
      )}
    </div>
  );
}
