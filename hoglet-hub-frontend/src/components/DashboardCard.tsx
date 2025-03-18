import Link from "next/link";
import React from "react";

export type AccentColor =
  | "blue"
  | "green"
  | "purple"
  | "orange"
  | "teal"
  | "red";

export interface DashboardCardProps {
  title: string;
  description: string;
  icon: string;
  link: string;
  disabled?: boolean;
  comingSoon?: boolean;
  accentColor?: AccentColor;
}

export function DashboardCard({
  title,
  description,
  icon,
  link,
  disabled = false,
  comingSoon = false,
  accentColor = "blue",
}: DashboardCardProps): React.ReactElement {
  const getAccentColorClass = (
    type: "border" | "text" | "description" | "badge"
  ): string => {
    const colorMap = {
      blue: {
        border: "border-t-4 border-blue-500",
        text: "text-blue-800",
        description: "text-blue-700 text-opacity-70",
        badge: "bg-blue-100 text-blue-800",
      },
      green: {
        border: "border-t-4 border-green-500",
        text: "text-green-800",
        description: "text-green-700 text-opacity-70",
        badge: "bg-green-100 text-green-800",
      },
      purple: {
        border: "border-t-4 border-purple-500",
        text: "text-purple-800",
        description: "text-purple-700 text-opacity-70",
        badge: "bg-purple-100 text-purple-800",
      },
      orange: {
        border: "border-t-4 border-orange-500",
        text: "text-orange-800",
        description: "text-orange-700 text-opacity-70",
        badge: "bg-orange-100 text-orange-800",
      },
      teal: {
        border: "border-t-4 border-teal-500",
        text: "text-teal-800",
        description: "text-teal-700 text-opacity-70",
        badge: "bg-teal-100 text-teal-800",
      },
      red: {
        border: "border-t-4 border-red-500",
        text: "text-red-800",
        description: "text-red-700 text-opacity-70",
        badge: "bg-red-100 text-red-800",
      },
    };

    return colorMap[accentColor][type];
  };

  const content = (
    <div
      className={`bg-gradient-to-b from-white to-gray-50 p-6 rounded-lg shadow-md hover:shadow-lg transition-all duration-300 ${getAccentColorClass(
        "border"
      )} ${disabled ? "opacity-70" : "hover:translate-y-[-2px]"}`}
      role={disabled ? "presentation" : "button"}
      aria-disabled={disabled}
    >
      <div className="flex items-center mb-4">
        <span className="text-3xl mr-3" aria-hidden="true">
          {icon}
        </span>
        <h2
          className={`text-xl font-semibold ${getAccentColorClass(
            "text"
          )} border-b border-opacity-20 pb-1`}
        >
          {title}
        </h2>
        {comingSoon && (
          <span
            className={`ml-2 px-2 py-1 text-xs rounded-full ${getAccentColorClass(
              "badge"
            )}`}
          >
            Coming Soon
          </span>
        )}
      </div>
      <p className={`${getAccentColorClass("description")} mb-4`}>
        {description}
      </p>
    </div>
  );

  if (disabled) {
    return content;
  }

  return (
    <Link href={link} aria-label={`Go to ${title}`}>
      {content}
    </Link>
  );
}
