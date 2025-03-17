"use client";

import { useAuth } from "../../contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import Link from "next/link";

export default function DashboardPage() {
  const { isAuthenticated } = useAuth();
  const router = useRouter();

  // Redirect to login if not authenticated
  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, router]);

  return (
    <div className="max-w-6xl mx-auto">
      <h1 className="text-3xl font-bold mb-8 text-center">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <DashboardCard
          title="Create Tenant"
          description="Provision a new tenant with custom settings."
          icon="💡"
          link="/tenants/create"
        />
        <DashboardCard
          title="View Operations"
          description="Track the status of ongoing and completed operations."
          icon="📊"
          link="/operations"
        />
        <DashboardCard
          title="System Status"
          description="Check the health and status of the platform components."
          icon="🔍"
          link="#"
          disabled={true}
          comingSoon={true}
        />
        <DashboardCard
          title="Manage Users"
          description="Add and manage users with different permission levels."
          icon="👥"
          link="#"
          disabled={true}
          comingSoon={true}
        />
        <DashboardCard
          title="Settings"
          description="Configure platform-wide settings and defaults."
          icon="⚙️"
          link="#"
          disabled={true}
          comingSoon={true}
        />
        <DashboardCard
          title="Help & Support"
          description="Get help and support for platform-related issues."
          icon="❓"
          link="#"
          disabled={true}
          comingSoon={true}
        />
      </div>

      <div className="mt-12 p-6 bg-blue-50 rounded-lg">
        <h2 className="text-xl font-semibold mb-4 text-blue-800">
          Recent Activity
        </h2>
        <p className="text-gray-600">
          Activity feed will be available in a future update. Stay tuned!
        </p>
      </div>
    </div>
  );
}

interface DashboardCardProps {
  title: string;
  description: string;
  icon: string;
  link: string;
  disabled?: boolean;
  comingSoon?: boolean;
}

function DashboardCard({
  title,
  description,
  icon,
  link,
  disabled = false,
  comingSoon = false,
}: DashboardCardProps) {
  const content = (
    <div
      className={`bg-white p-6 rounded-lg shadow-md hover:shadow-lg transition-shadow ${
        disabled ? "opacity-70" : ""
      }`}
    >
      <div className="flex items-center mb-4">
        <span className="text-3xl mr-3">{icon}</span>
        <h2 className="text-xl font-semibold">{title}</h2>
        {comingSoon && (
          <span className="ml-2 px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded-full">
            Coming Soon
          </span>
        )}
      </div>
      <p className="text-gray-600 mb-4">{description}</p>
    </div>
  );

  if (disabled) {
    return content;
  }

  return <Link href={link}>{content}</Link>;
}
