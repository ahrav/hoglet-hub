import Link from "next/link";
import React from "react";

export interface FeatureCardProps {
  title: string;
  description: string;
  linkHref: string;
  linkText: string;
}

export function FeatureCard({
  title,
  description,
  linkHref,
  linkText,
}: FeatureCardProps): React.ReactElement {
  return (
    <article className="bg-white p-6 rounded-lg shadow-md hover:shadow-lg transition-shadow">
      <h2 className="text-2xl font-semibold mb-3 text-blue-600">{title}</h2>
      <p className="text-gray-600 mb-4">{description}</p>
      <Link
        href={linkHref}
        className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors"
      >
        {linkText}
      </Link>
    </article>
  );
}
