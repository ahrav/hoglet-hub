import React from "react";
import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "../contexts/AuthContext";
import { QueryProvider } from "../providers/QueryProvider";
import { ThemeProvider } from "../contexts/ThemeContext";
import PageLayout from "../components/layout/PageLayout";
// This import is required for its side effects - it initializes the API configuration
// at startup by immediately executing the initializeApi function.
// TODO: Try to figure out a real solution... Jerbilsss
import "../api/config";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "Hoglet Hub - Tenant Provisioning",
    template: "%s | Hoglet Hub",
  },
  description:
    "Internal tenant provisioning system for managing and provisioning tenants across multiple regions",
  viewport: "width=device-width, initial-scale=1",
  robots: "noindex, nofollow", // Since it's an internal tool
};

function AppProviders({
  children,
}: {
  children: React.ReactNode;
}): React.ReactElement {
  return (
    <QueryProvider>
      <ThemeProvider>
        <AuthProvider>{children}</AuthProvider>
      </ThemeProvider>
    </QueryProvider>
  );
}

/**
 * Root layout component that wraps the entire application
 */
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>): React.ReactElement {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {/*
          TODO: Consider adding an ErrorBoundary component for better error handling
        */}
        <AppProviders>
          <PageLayout>{children}</PageLayout>
        </AppProviders>
      </body>
    </html>
  );
}
