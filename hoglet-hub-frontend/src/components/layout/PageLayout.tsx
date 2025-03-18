"use client";

import { ReactNode, useMemo } from "react";
import Navbar from "./Navbar";

// Layout constants
const COPYRIGHT_PREFIX = "© ";
const COPYRIGHT_SUFFIX = " Hoglet Hub. All rights reserved.";
const MAIN_BASE_CLASSES =
  "flex-grow container mx-auto px-4 py-8 dark:text-gray-100";
const LAYOUT_CONTAINER_CLASSES =
  "min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900";
const FOOTER_CLASSES = "bg-gray-800 dark:bg-gray-950 text-white py-4";
const FOOTER_CONTENT_CLASSES = "container mx-auto px-4 text-center";

interface PageLayoutProps {
  children: ReactNode;
  hideFooter?: boolean;
  mainClassName?: string;
}

export default function PageLayout({
  children,
  hideFooter = false,
  mainClassName = "",
}: PageLayoutProps) {
  const currentYear = useMemo(() => new Date().getFullYear(), []);

  const mainClasses = `${MAIN_BASE_CLASSES} ${mainClassName}`;

  return (
    <div className={LAYOUT_CONTAINER_CLASSES}>
      <header>
        <Navbar />
      </header>

      <main className={mainClasses} id="main-content" tabIndex={-1} role="main">
        {children}
      </main>

      {!hideFooter && (
        <footer className={FOOTER_CLASSES} role="contentinfo">
          <div className={FOOTER_CONTENT_CLASSES}>
            <p>
              {COPYRIGHT_PREFIX}
              {currentYear}
              {COPYRIGHT_SUFFIX}
            </p>
          </div>
        </footer>
      )}
    </div>
  );
}
