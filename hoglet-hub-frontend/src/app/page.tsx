import Link from "next/link";

export default function Home() {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <h1 className="text-4xl font-bold text-center mb-6">
        Welcome to Hoglet Hub
      </h1>
      <p className="text-xl text-gray-600 text-center max-w-2xl mb-8">
        Our internal platform for provisioning and managing tenants across
        multiple regions.
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 w-full max-w-4xl">
        <div className="bg-white p-6 rounded-lg shadow-md hover:shadow-lg transition-shadow">
          <h2 className="text-2xl font-semibold mb-3 text-blue-600">
            Create New Tenant
          </h2>
          <p className="text-gray-600 mb-4">
            Provision a new tenant instance in your preferred region with
            customizable settings.
          </p>
          <Link
            href="/tenants/create"
            className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors"
          >
            Get Started
          </Link>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-md hover:shadow-lg transition-shadow">
          <h2 className="text-2xl font-semibold mb-3 text-blue-600">
            Monitor Operations
          </h2>
          <p className="text-gray-600 mb-4">
            Track the status of ongoing operations and view detailed execution
            history.
          </p>
          <Link
            href="/operations"
            className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors"
          >
            View Operations
          </Link>
        </div>
      </div>

      <div className="mt-12 bg-blue-50 p-6 rounded-lg max-w-4xl">
        <h2 className="text-2xl font-semibold mb-3 text-blue-600">
          System Status
        </h2>
        <div className="flex items-center">
          <div className="w-3 h-3 bg-green-500 rounded-full mr-2"></div>
          <p className="text-gray-700">All systems operational</p>
        </div>
      </div>
    </div>
  );
}
