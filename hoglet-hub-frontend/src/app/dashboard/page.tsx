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
          accentColor="green"
        />
        <DashboardCard
          title="View Operations"
          description="Track the status of ongoing and completed operations."
          icon="📊"
          link="/operations"
          accentColor="blue"
        />
        <DashboardCard
          title="System Status"
          description="Check the health and status of the platform components."
          icon="🔍"
          link="#"
          disabled={true}
          comingSoon={true}
          accentColor="purple"
        />
        <DashboardCard
          title="Manage Users"
          description="Add and manage users with different permission levels."
          icon="👥"
          link="#"
          disabled={true}
          comingSoon={true}
          accentColor="orange"
        />
        <DashboardCard
          title="Settings"
          description="Configure platform-wide settings and defaults."
          icon="⚙️"
          link="#"
          disabled={true}
          comingSoon={true}
          accentColor="teal"
        />
        <DashboardCard
          title="Help & Support"
          description="Get help and support for platform-related issues."
          icon="❓"
          link="#"
          disabled={true}
          comingSoon={true}
          accentColor="red"
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
  accentColor?: "blue" | "green" | "purple" | "orange" | "teal" | "red";
}

function DashboardCard({
  title,
  description,
  icon,
  link,
  disabled = false,
  comingSoon = false,
  accentColor = "blue",
}: DashboardCardProps) {
  const accentColorClasses = {
    blue: "border-t-4 border-blue-500",
    green: "border-t-4 border-green-500",
    purple: "border-t-4 border-purple-500",
    orange: "border-t-4 border-orange-500",
    teal: "border-t-4 border-teal-500",
    red: "border-t-4 border-red-500",
  };

  const titleColorClasses = {
    blue: "text-blue-800",
    green: "text-green-800",
    purple: "text-purple-800",
    orange: "text-orange-800",
    teal: "text-teal-800",
    red: "text-red-800",
  };

  const descriptionColorClasses = {
    blue: "text-blue-700 text-opacity-70",
    green: "text-green-700 text-opacity-70",
    purple: "text-purple-700 text-opacity-70",
    orange: "text-orange-700 text-opacity-70",
    teal: "text-teal-700 text-opacity-70",
    red: "text-red-700 text-opacity-70",
  };

  const badgeColorClasses = {
    blue: "bg-blue-100 text-blue-800",
    green: "bg-green-100 text-green-800",
    purple: "bg-purple-100 text-purple-800",
    orange: "bg-orange-100 text-orange-800",
    teal: "bg-teal-100 text-teal-800",
    red: "bg-red-100 text-red-800",
  };

  const content = (
    <div
      className={`bg-gradient-to-b from-white to-gray-50 p-6 rounded-lg shadow-md hover:shadow-lg transition-all duration-300 ${
        accentColorClasses[accentColor]
      } ${disabled ? "opacity-70" : "hover:translate-y-[-2px]"}`}
    >
      <div className="flex items-center mb-4">
        <span className="text-3xl mr-3">{icon}</span>
        <h2
          className={`text-xl font-semibold ${titleColorClasses[accentColor]} border-b border-opacity-20 pb-1`}
        >
          {title}
        </h2>
        {comingSoon && (
          <span
            className={`ml-2 px-2 py-1 text-xs rounded-full ${badgeColorClasses[accentColor]}`}
          >
            Coming Soon
          </span>
        )}
      </div>
      <p className={`${descriptionColorClasses[accentColor]} mb-4`}>
        {description}
      </p>
    </div>
  );

  if (disabled) {
    return content;
  }

  return <Link href={link}>{content}</Link>;
}
