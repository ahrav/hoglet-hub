"use client";

import { ReactNode, useMemo } from "react";
import Navbar from "./Navbar";

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

  const mainClasses = `flex-grow container mx-auto px-4 py-8 dark:text-gray-100 ${mainClassName}`;

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <header>
        <Navbar />
      </header>

      <main className={mainClasses} id="main-content" tabIndex={-1} role="main">
        {children}
      </main>

      {!hideFooter && (
        <footer
          className="bg-gray-800 dark:bg-gray-950 text-white py-4"
          role="contentinfo"
        >
          <div className="container mx-auto px-4 text-center">
            <p>© {currentYear} Hoglet Hub. All rights reserved.</p>
          </div>
        </footer>
      )}
    </div>
  );
}
