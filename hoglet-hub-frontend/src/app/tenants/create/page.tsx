"use client";

import React, { useEffect, useState } from "react";
import TenantCreateForm from "../../../components/TenantCreateForm";
import { useAuth } from "../../../contexts/AuthContext";
import { useRouter } from "next/navigation";

// Page constants
const PAGE_TITLE = "Create New Tenant";
const LOADING_MESSAGE = "Loading...";
const LOADING_CONTAINER_CLASSES =
  "flex justify-center items-center min-h-screen";
const LOADING_TEXT_CLASSES = "text-xl text-gray-600";

export default function CreateTenantPage(): React.ReactElement {
  const { isAuthenticated } = useAuth();
  const router = useRouter();
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);

  useEffect(() => {
    if (isAuthenticated === false) {
      router.push("/login");
    } else if (isAuthenticated === true) {
      setIsCheckingAuth(false);
    }
  }, [isAuthenticated, router]);

  if (isCheckingAuth) {
    return (
      <div className={LOADING_CONTAINER_CLASSES}>
        <div className={LOADING_TEXT_CLASSES}>{LOADING_MESSAGE}</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center">
      <div className="w-full max-w-3xl">
        <h1 className="text-3xl font-bold mb-8 text-center">{PAGE_TITLE}</h1>
        <TenantCreateForm />
      </div>
    </div>
  );
}
