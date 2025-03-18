"use client";

import React from "react";
import LoginForm from "../../components/LoginForm";

// Layout constants
const CONTAINER_CLASSES = "flex justify-center items-center py-12";
const FORM_CONTAINER_CLASSES = "w-full max-w-md";

export default function LoginPage(): React.ReactElement {
  return (
    <div className={CONTAINER_CLASSES}>
      <div className={FORM_CONTAINER_CLASSES}>
        <LoginForm />
      </div>
    </div>
  );
}
