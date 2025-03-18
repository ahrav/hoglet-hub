"use client";

import React, { useEffect, useState } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useRouter } from "next/navigation";
import {
  DashboardCard,
  DashboardCardProps,
} from "../../components/DashboardCard";

// Page constants
const PAGE_TITLE = "Dashboard";
const ACTIVITY_SECTION_TITLE = "Recent Activity";
const ACTIVITY_PLACEHOLDER =
  "Activity feed will be available in a future update. Stay tuned!";
const LOADING_MESSAGE = "Loading...";
const LOADING_CONTAINER_CLASSES =
  "flex justify-center items-center min-h-screen";
const LOADING_TEXT_CLASSES = "text-xl text-gray-600";

// Dashboard cards data
const DASHBOARD_CARDS: DashboardCardProps[] = [
  {
    title: "Create Tenant",
    description: "Provision a new tenant with custom settings.",
    icon: "💡",
    link: "/tenants/create",
    accentColor: "green",
  },
  {
    title: "View Operations",
    description: "Track the status of ongoing and completed operations.",
    icon: "📊",
    link: "/operations",
    accentColor: "blue",
  },
  {
    title: "System Status",
    description: "Check the health and status of the platform components.",
    icon: "🔍",
    link: "#",
    disabled: true,
    comingSoon: true,
    accentColor: "purple",
  },
  {
    title: "Manage Users",
    description: "Add and manage users with different permission levels.",
    icon: "👥",
    link: "#",
    disabled: true,
    comingSoon: true,
    accentColor: "orange",
  },
  {
    title: "Settings",
    description: "Configure platform-wide settings and defaults.",
    icon: "⚙️",
    link: "#",
    disabled: true,
    comingSoon: true,
    accentColor: "teal",
  },
  {
    title: "Help & Support",
    description: "Get help and support for platform-related issues.",
    icon: "❓",
    link: "#",
    disabled: true,
    comingSoon: true,
    accentColor: "red",
  },
];

export default function DashboardPage(): React.ReactElement {
  const { isAuthenticated } = useAuth();
  const router = useRouter();
  const [isCheckingAuth, setIsCheckingAuth] = useState<boolean>(true);

  // Authentication check and redirect
  useEffect(() => {
    if (isAuthenticated === false) {
      router.push("/login");
    } else if (isAuthenticated === true) {
      setIsCheckingAuth(false);
    }
  }, [isAuthenticated, router]);

  // Show loading state while checking authentication
  if (isCheckingAuth) {
    return (
      <div className={LOADING_CONTAINER_CLASSES}>
        <div className={LOADING_TEXT_CLASSES}>{LOADING_MESSAGE}</div>
      </div>
    );
  }

  return (
    <main className="max-w-6xl mx-auto" role="main">
      <h1 className="text-3xl font-bold mb-8 text-center">{PAGE_TITLE}</h1>

      <section
        aria-label="Dashboard Actions"
        className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
      >
        {DASHBOARD_CARDS.map((card, index) => (
          <DashboardCard key={index} {...card} />
        ))}
      </section>

      <section
        aria-label={ACTIVITY_SECTION_TITLE}
        className="mt-12 p-6 bg-blue-50 rounded-lg"
      >
        <h2 className="text-xl font-semibold mb-4 text-blue-800">
          {ACTIVITY_SECTION_TITLE}
        </h2>
        <p className="text-gray-600">{ACTIVITY_PLACEHOLDER}</p>
      </section>
    </main>
  );
}
