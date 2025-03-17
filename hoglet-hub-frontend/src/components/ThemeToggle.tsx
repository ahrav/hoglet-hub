"use client";

import { useTheme } from "@/contexts/ThemeContext";
import { useCallback, useEffect, useState } from "react";

// SVG components to improve readability
const SunIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="16"
    height="16"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    role="img"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="5"></circle>
    <line x1="12" y1="1" x2="12" y2="3"></line>
    <line x1="12" y1="21" x2="12" y2="23"></line>
    <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
    <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
    <line x1="1" y1="12" x2="3" y2="12"></line>
    <line x1="21" y1="12" x2="23" y2="12"></line>
    <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
    <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
  </svg>
);

const MoonIcon = () => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="16"
    height="16"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    role="img"
    aria-hidden="true"
  >
    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
  </svg>
);

export default function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  // Get effective theme (light/dark) accounting for system setting
  const [effectiveTheme, setEffectiveTheme] = useState<'light' | 'dark'>(
    theme === 'system' ? 'light' : theme
  );

  // Update effective theme when needed
  useEffect(() => {
    if (theme !== 'system') {
      setEffectiveTheme(theme);
      return;
    }

    // For system theme, determine from media query
    const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    setEffectiveTheme(isDark ? 'dark' : 'light');

    // Listen for changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      setEffectiveTheme(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme]);

  // Avoid hydration mismatch by only rendering after mount
  useEffect(() => {
    setMounted(true);
  }, []);

  // Memoized event handlers
  const handleSelectChange = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    if (value === 'light' || value === 'dark' || value === 'system') {
      setTheme(value);
    }
  }, [setTheme]);

  const handleToggleClick = useCallback(() => {
    if (theme === 'system') {
      // When toggling from system, switch to the opposite of current effective theme
      setTheme(effectiveTheme === 'dark' ? 'light' : 'dark');
    } else {
      // Otherwise toggle between light and dark
      setTheme(theme === 'dark' ? 'light' : 'dark');
    }
  }, [theme, effectiveTheme, setTheme]);

  if (!mounted) return null;

  return (
    <div className="flex items-center space-x-2">
      <select
        value={theme}
        onChange={handleSelectChange}
        className="bg-transparent border rounded px-2 py-1 text-sm"
        aria-label="Select theme"
      >
        <option value="light">Light</option>
        <option value="dark">Dark</option>
        <option value="system">System</option>
      </select>

      {/* Quick toggle button */}
      <button
        onClick={handleToggleClick}
        className="p-2 rounded-full bg-gray-200 dark:bg-gray-700"
        aria-label={`Switch to ${effectiveTheme === "dark" ? "light" : "dark"} mode`}
      >
        {effectiveTheme === "dark" ? <SunIcon /> : <MoonIcon />}
      </button>
    </div>
  );
}
