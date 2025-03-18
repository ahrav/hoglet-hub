import React from "react";
import Link from "next/link";
import { FeatureCard } from "../components/FeatureCard";

export const metadata = {
  title: 'Hoglet Hub - Internal Management Platform',
  description: 'Our internal platform for provisioning and managing tenants across multiple regions.'
};

// TODO: Move this to proper API service once added to OpenAPI spec
async function getSystemStatus() {
  // Temporary mock implementation
  return {
    isOperational: true,
    message: "All systems operational"
  };
}

export default async function Home(): Promise<React.ReactElement> {
  const status = await getSystemStatus();

  return (
    <main className="flex flex-col items-center justify-center py-12">
      <h1 className="text-4xl font-bold text-center mb-6">
        Welcome to Hoglet Hub
      </h1>
      <p className="text-xl text-gray-600 text-center max-w-2xl mb-8">
        Our internal platform for provisioning and managing tenants across
        multiple regions.
      </p>

      <section className="grid grid-cols-1 md:grid-cols-2 gap-6 w-full max-w-4xl">
        <FeatureCard
          title="Create New Tenant"
          description="Provision a new tenant instance in your preferred region with customizable settings."
          linkHref="/tenants/create"
          linkText="Get Started"
        />

        <FeatureCard
          title="Monitor Operations"
          description="Track the status of ongoing operations and view detailed execution history."
          linkHref="/operations"
          linkText="View Operations"
        />
      </section>

      <section aria-label="System Status" className="mt-12 bg-blue-50 p-6 rounded-lg max-w-4xl">
        <h2 className="text-2xl font-semibold mb-3 text-blue-600">
          System Status
        </h2>
        <div className="flex items-center">
          <div
            className={`w-3 h-3 ${status.isOperational ? 'bg-green-500' : 'bg-red-500'} rounded-full mr-2`}
            aria-hidden="true"
          ></div>
          <p className="text-gray-700">{status.message}</p>
        </div>
      </section>
    </main>
  );
}
