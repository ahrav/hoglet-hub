"use client";

import { useState } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useOperationsApi } from "../../hooks/useOperationsApi";
import OperationDetails from "../../components/OperationDetails";

export default function OperationsPage() {
  const { isAuthenticated } = useAuth();
  const router = useRouter();
  const { getOperation, operation, isLoading, error } = useOperationsApi();
  const [operationId, setOperationId] = useState<string>("");

  // Redirect to login if not authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const id = parseInt(operationId.trim());
    if (!isNaN(id)) {
      await getOperation(id);
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <h1 className="text-3xl font-bold mb-8 text-center">View Operations</h1>

      <div className="bg-white rounded-lg shadow-md p-6 mb-8">
        <h2 className="text-xl font-semibold mb-4">Lookup an Operation</h2>
        <form onSubmit={handleSubmit} className="flex gap-4">
          <input
            type="text"
            value={operationId}
            onChange={(e) => setOperationId(e.target.value)}
            placeholder="Enter Operation ID"
            className="flex-grow px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-gray-400"
          >
            {isLoading ? "Loading..." : "View"}
          </button>
        </form>

        {error && (
          <div className="mt-4 p-3 bg-red-100 text-red-700 rounded-md">
            <p>{error.message}</p>
          </div>
        )}
      </div>

      {operation && <OperationDetails operation={operation} />}

      {!operation && !isLoading && !error && (
        <div className="bg-gray-50 rounded-lg p-8 text-center">
          <p className="text-gray-500 text-lg">
            Enter an operation ID to view its details
          </p>
        </div>
      )}
    </div>
  );
}
