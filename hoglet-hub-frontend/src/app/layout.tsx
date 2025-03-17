import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "../contexts/AuthContext";
import { QueryProvider } from "../providers/QueryProvider";
import { ThemeProvider } from "../contexts/ThemeContext";
import PageLayout from "../components/layout/PageLayout";
// Import API configuration to ensure it's loaded at startup
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
  title: "Hoglet Hub - Tenant Provisioning",
  description: "Internal tenant provisioning system",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <QueryProvider>
          <ThemeProvider>
            <AuthProvider>
              <PageLayout>{children}</PageLayout>
            </AuthProvider>
          </ThemeProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
