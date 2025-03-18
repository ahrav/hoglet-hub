"use client";

import React from "react";
import LoginForm from "../../components/LoginForm";

export default function LoginPage(): React.ReactElement {
  return (
    <div className="flex justify-center items-center py-12">
      <div className="w-full max-w-md">
        <LoginForm />
      </div>
    </div>
  );
}
