import { DefaultService } from "./generated/services/DefaultService";
import { OpenAPI } from "./generated/core/OpenAPI";
import { TenantCreate } from "./generated/models/TenantCreate";
import { API_BASE_URL } from "./config";
import { CancelablePromise } from "./generated/core/CancelablePromise";

// Set the correct base URL before any API calls
// This ensures the OpenAPI configuration is properly set
OpenAPI.BASE = API_BASE_URL;
OpenAPI.WITH_CREDENTIALS = true;

// Utility function to create cancellable versions of API calls
function withCancellation<T>(promiseFactory: () => CancelablePromise<T>) {
  const promise = promiseFactory();
  return {
    promise,
    cancel: () => promise.cancel(),
  };
}

export const TenantService = {
  createTenant: async (params: { requestBody: TenantCreate }) => {
    try {
      return await DefaultService.createTenant(params);
    } catch (error) {
      console.error("Failed to create tenant:", error);
      throw error;
    }
  },

  createTenantWithCancellation: (params: { requestBody: TenantCreate }) =>
    withCancellation(() => DefaultService.createTenant(params)),

  deleteTenant: async (params: { tenantId: number }) => {
    try {
      return await DefaultService.deleteTenant(params);
    } catch (error) {
      console.error(`Failed to delete tenant ${params.tenantId}:`, error);
      throw error;
    }
  },

  deleteTenantWithCancellation: (params: { tenantId: number }) =>
    withCancellation(() => DefaultService.deleteTenant(params)),
};

export const OperationService = {
  getOperation: async (params: { operationId: number }) => {
    try {
      return await DefaultService.getOperation(params);
    } catch (error) {
      console.error(`Failed to get operation ${params.operationId}:`, error);
      throw error;
    }
  },

  getOperationWithCancellation: (params: { operationId: number }) =>
    withCancellation(() => DefaultService.getOperation(params)),
};

export { DefaultService };
