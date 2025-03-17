import { DefaultService } from "./generated/services/DefaultService";
import { OpenAPI } from "./generated/core/OpenAPI";
import { TenantCreate } from "./generated/models/TenantCreate";
import { API_BASE_URL } from "./config";

// Set the correct base URL before any API calls
// This ensures the OpenAPI configuration is properly set
OpenAPI.BASE = API_BASE_URL;
OpenAPI.WITH_CREDENTIALS = true;

// Create wrapper functions around DefaultService methods to ensure
// the BASE URL is set correctly before each call
export const TenantService = {
  async createTenant(params: { requestBody: TenantCreate }) {
    // Ensure BASE URL is set correctly before each API call
    OpenAPI.BASE = API_BASE_URL;
    return DefaultService.createTenant(params);
  },

  async deleteTenant(params: { tenantId: number }) {
    OpenAPI.BASE = API_BASE_URL;
    return DefaultService.deleteTenant(params);
  },
};

export const OperationService = {
  async getOperation(params: { operationId: number }) {
    OpenAPI.BASE = API_BASE_URL;
    return DefaultService.getOperation(params);
  },
};

// Export the original service for direct access
// (though using the wrappers above is preferred)
export { DefaultService };
