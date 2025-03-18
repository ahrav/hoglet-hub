"use client";

import { useState, useCallback, useMemo } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "../../contexts/AuthContext";
import ThemeToggle from "../ThemeToggle";

// Navigation constants
const BRAND_NAME = "Hoglet Hub";
const LOGIN_TEXT = "Login";
const LOGOUT_TEXT = "Logout";
const LOGOUT_LOADING_TEXT = "Logging out...";
const MENU_TOGGLE_LABEL = "Open main menu";

interface NavItem {
  href: string;
  label: string;
  requiresAuth: boolean;
}

export default function Navbar() {
  const { isAuthenticated, logout } = useAuth();
  const pathname = usePathname();
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  const toggleMenu = useCallback(() => {
    setIsMenuOpen((prev) => !prev);
  }, []);

  const navItems = useMemo<NavItem[]>(
    () => [
      { href: "/", label: "Home", requiresAuth: false },
      { href: "/dashboard", label: "Dashboard", requiresAuth: true },
      { href: "/tenants/create", label: "Create Tenant", requiresAuth: true },
    ],
    []
  );

  // Filter nav items based on auth status
  const filteredNavItems = useMemo(
    () => navItems.filter((item) => !item.requiresAuth || isAuthenticated),
    [navItems, isAuthenticated]
  );

  const handleLogout = useCallback(async () => {
    try {
      setIsLoggingOut(true);
      logout();
    } catch (error) {
      console.error("Logout failed:", error);
    } finally {
      setIsLoggingOut(false);
    }
  }, [logout]);

  return (
    <nav
      className="bg-blue-600 dark:bg-blue-800 text-white shadow-md"
      aria-label="Main navigation"
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link
              href="/"
              className="flex items-center"
              aria-label={`${BRAND_NAME} home`}
            >
              <span className="text-xl font-bold">{BRAND_NAME}</span>
            </Link>
          </div>

          {/* Desktop menu */}
          <div className="hidden md:flex items-center space-x-4">
            {filteredNavItems.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={`px-3 py-2 rounded-md text-sm font-medium
                  ${
                    pathname === item.href
                      ? "bg-blue-700 dark:bg-blue-600"
                      : "hover:bg-blue-700 dark:hover:bg-blue-600"
                  }
                  transition-colors`}
                aria-current={pathname === item.href ? "page" : undefined}
              >
                {item.label}
              </Link>
            ))}

            {isAuthenticated ? (
              <button
                onClick={handleLogout}
                disabled={isLoggingOut}
                className={`px-3 py-2 rounded-md text-sm font-medium
                  hover:bg-blue-700 dark:hover:bg-blue-600 transition-colors
                  ${isLoggingOut ? "opacity-70 cursor-not-allowed" : ""}`}
                aria-busy={isLoggingOut}
              >
                {isLoggingOut ? LOGOUT_LOADING_TEXT : LOGOUT_TEXT}
              </button>
            ) : (
              <Link
                href="/login"
                className={`px-3 py-2 rounded-md text-sm font-medium
                  ${
                    pathname === "/login"
                      ? "bg-blue-700 dark:bg-blue-600"
                      : "hover:bg-blue-700 dark:hover:bg-blue-600"
                  }
                  transition-colors`}
                aria-current={pathname === "/login" ? "page" : undefined}
              >
                {LOGIN_TEXT}
              </Link>
            )}

            <ThemeToggle />
          </div>

          {/* Mobile menu button */}
          <div className="md:hidden flex items-center">
            <button
              type="button"
              className="inline-flex items-center justify-center p-2 rounded-md text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-white"
              aria-controls="mobile-menu"
              aria-expanded={isMenuOpen}
              onClick={toggleMenu}
            >
              <span className="sr-only">{MENU_TOGGLE_LABEL}</span>
              {/* Icon for menu open/close */}
              <svg
                className={`h-6 w-6 ${isMenuOpen ? "hidden" : "block"}`}
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
              <svg
                className={`h-6 w-6 ${isMenuOpen ? "block" : "hidden"}`}
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Mobile menu, show/hide based on menu state */}
      <div
        className={`md:hidden ${isMenuOpen ? "block" : "hidden"}`}
        id="mobile-menu"
      >
        <div className="px-2 pt-2 pb-3 space-y-1 sm:px-3">
          {filteredNavItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`block px-3 py-2 rounded-md text-base font-medium
                ${
                  pathname === item.href
                    ? "bg-blue-700 dark:bg-blue-600"
                    : "hover:bg-blue-700 dark:hover:bg-blue-600"
                }
                transition-colors`}
              aria-current={pathname === item.href ? "page" : undefined}
              onClick={() => setIsMenuOpen(false)} // Close menu when link clicked
            >
              {item.label}
            </Link>
          ))}

          {isAuthenticated ? (
            <button
              onClick={() => {
                handleLogout();
                setIsMenuOpen(false);
              }}
              disabled={isLoggingOut}
              className={`w-full text-left block px-3 py-2 rounded-md text-base font-medium
                hover:bg-blue-700 dark:hover:bg-blue-600 transition-colors
                ${isLoggingOut ? "opacity-70 cursor-not-allowed" : ""}`}
              aria-busy={isLoggingOut}
            >
              {isLoggingOut ? LOGOUT_LOADING_TEXT : LOGOUT_TEXT}
            </button>
          ) : (
            <Link
              href="/login"
              className={`block px-3 py-2 rounded-md text-base font-medium
                ${
                  pathname === "/login"
                    ? "bg-blue-700 dark:bg-blue-600"
                    : "hover:bg-blue-700 dark:hover:bg-blue-600"
                }
                transition-colors`}
              aria-current={pathname === "/login" ? "page" : undefined}
              onClick={() => setIsMenuOpen(false)}
            >
              {LOGIN_TEXT}
            </Link>
          )}

          <div className="px-3 py-2">
            <ThemeToggle />
          </div>
        </div>
      </div>
    </nav>
  );
}
