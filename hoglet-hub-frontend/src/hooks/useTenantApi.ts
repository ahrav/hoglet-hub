"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { TenantCreate } from "../api/generated/models/TenantCreate";
import { TenantService, OperationService } from "../api/services";
import { useAuth } from "../contexts/AuthContext";
import { OperationResponse } from "../api/generated/models/OperationResponse";

export function useTenantApi() {
  const { isAuthenticated } = useAuth();

  // Create tenant mutation
  const createTenantMutation = useMutation({
    mutationFn: (tenantData: TenantCreate) =>
      TenantService.createTenant({ requestBody: tenantData }),
    onError: (error) => {
      console.error("Failed to create tenant:", error);
    },
  });

  // Delete tenant mutation
  const deleteTenantMutation = useMutation({
    mutationFn: (tenantId: number) => TenantService.deleteTenant({ tenantId }),
    onError: (error) => {
      console.error("Failed to delete tenant:", error);
    },
  });

  // Get operation query factory with dynamic polling.
  const getOperation = (operationId: number | null) => {
    return useQuery<
      OperationResponse | null,
      Error,
      (OperationResponse & { isComplete: boolean }) | null,
      [string, number | null]
    >({
      queryKey: ["operation", operationId],
      queryFn: () =>
        operationId
          ? OperationService.getOperation({ operationId })
          : Promise.resolve(null),
      enabled: !!operationId && isAuthenticated,
      refetchInterval: (query) => {
        const { data } = query.state;
        if (
          data &&
          ["completed", "failed", "cancelled"].includes(data.status)
        ) {
          return false; // Stop polling when terminal state is reached
        }
        return 2000; // Otherwise poll every 2 seconds
      },
      refetchIntervalInBackground: true,
      staleTime: 1000,
      refetchOnWindowFocus: false,
      select: (data) => {
        if (!data) return null;
        return {
          ...data,
          isComplete: ["completed", "failed", "cancelled"].includes(
            data.status
          ),
        } as OperationResponse & { isComplete: boolean };
      },
    });
  };

  return {
    createTenant: createTenantMutation,
    deleteTenant: deleteTenantMutation,
    getOperation,
  };
}
