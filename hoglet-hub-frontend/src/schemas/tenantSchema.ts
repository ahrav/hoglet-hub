import { z } from "zod";

// Constants for reuse and type safety
export const REGIONS = [
  "us1",
  "us2",
  "us3",
  "us4",
  "eu1",
  "eu2",
  "eu3",
  "eu4",
] as const;
export const TIERS = ["free", "pro", "enterprise"] as const;
export type Region = (typeof REGIONS)[number];
export type Tier = (typeof TIERS)[number];

// Reusable field definitions
export const nameSchema = z
  .string()
  .min(2, "Name must be at least 2 characters")
  .max(64, "Name cannot exceed 64 characters")
  .regex(
    /^[a-z0-9-]+$/,
    "Name can only contain lowercase letters, numbers, and hyphens"
  )
  .describe("Tenant identifier");

export const regionSchema = z
  .enum(REGIONS, {
    errorMap: () => ({ message: "Please select a valid region" }),
  })
  .describe("Deployment region");

export const tierSchema = z
  .enum(TIERS, {
    errorMap: () => ({ message: "Please select a valid tier" }),
  })
  .describe("Service tier");

// Create the base schema object without refinements
const baseTenantSchema = z.object({
  name: nameSchema,
  region: regionSchema,
  tier: tierSchema.default("free"),
  isolation_group_id: z.number().nullable().optional(),
});

// Apply the refinement function to check business logic.
// TODO: Revist, just testing stuff.
export const tenantCreateSchema = baseTenantSchema.refine(
  (data) => !(data.tier === "free" && data.isolation_group_id !== null),
  {
    message: "Free tier cannot specify an isolation group",
    path: ["isolation_group_id"],
  }
);

// Create the details schema by extending the base schema (not the refined one)
export const tenantDetailsSchema = baseTenantSchema.extend({
  id: z.number(),
  created_at: z.string().datetime(),
  updated_at: z.string().datetime().optional(),
  status: z.enum(["active", "inactive", "provisioning", "error"]),
});

// For update operations - make all fields optional from the base schema
export const tenantUpdateSchema = baseTenantSchema.partial();

// Infer the TypeScript types from the schemas
export type TenantCreateFormData = z.infer<typeof tenantCreateSchema>;
export type TenantDetails = z.infer<typeof tenantDetailsSchema>;
export type TenantUpdateFormData = z.infer<typeof tenantUpdateSchema>;
