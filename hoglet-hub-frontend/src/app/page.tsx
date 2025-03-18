import React from "react";
import Link from "next/link";
import { FeatureCard } from "../components/FeatureCard";

// Page constants
const PAGE_TITLE = "Hoglet Hub - Internal Management Platform";
const PAGE_DESCRIPTION =
  "Our internal platform for provisioning and managing tenants across multiple regions.";
const WELCOME_HEADING = "Welcome to Hoglet Hub";
const SYSTEM_STATUS_HEADING = "System Status";
const CREATE_TENANT_TITLE = "Create New Tenant";
const CREATE_TENANT_DESCRIPTION =
  "Provision a new tenant instance in your preferred region with customizable settings.";
const CREATE_TENANT_LINK_TEXT = "Get Started";
const MONITOR_OPERATIONS_TITLE = "Monitor Operations";
const MONITOR_OPERATIONS_DESCRIPTION =
  "Track the status of ongoing operations and view detailed execution history.";
const MONITOR_OPERATIONS_LINK_TEXT = "View Operations";

export const metadata = {
  title: PAGE_TITLE,
  description: PAGE_DESCRIPTION,
};

// TODO: Move this to proper API service once added to OpenAPI spec
async function getSystemStatus() {
  // Temporary mock implementation
  return {
    isOperational: true,
    message: "All systems operational",
  };
}

export default async function Home(): Promise<React.ReactElement> {
  const status = await getSystemStatus();

  return (
    <main className="flex flex-col items-center justify-center py-12">
      <h1 className="text-4xl font-bold text-center mb-6">{WELCOME_HEADING}</h1>
      <p className="text-xl text-gray-600 text-center max-w-2xl mb-8">
        {PAGE_DESCRIPTION}
      </p>

      <section className="grid grid-cols-1 md:grid-cols-2 gap-6 w-full max-w-4xl">
        <FeatureCard
          title={CREATE_TENANT_TITLE}
          description={CREATE_TENANT_DESCRIPTION}
          linkHref="/tenants/create"
          linkText={CREATE_TENANT_LINK_TEXT}
        />

        <FeatureCard
          title={MONITOR_OPERATIONS_TITLE}
          description={MONITOR_OPERATIONS_DESCRIPTION}
          linkHref="/operations"
          linkText={MONITOR_OPERATIONS_LINK_TEXT}
        />
      </section>

      <section
        aria-label="System Status"
        className="mt-12 bg-blue-50 p-6 rounded-lg max-w-4xl"
      >
        <h2 className="text-2xl font-semibold mb-3 text-blue-600">
          {SYSTEM_STATUS_HEADING}
        </h2>
        <div className="flex items-center">
          <div
            className={`w-3 h-3 ${
              status.isOperational ? "bg-green-500" : "bg-red-500"
            } rounded-full mr-2`}
            aria-hidden="true"
          ></div>
          <p className="text-gray-700">{status.message}</p>
        </div>
      </section>
    </main>
  );
}
