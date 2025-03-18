import React from "react";
import { useForm } from "react-hook-form";
import { useLoginForm, LoginFormData } from "../hooks/useLoginForm";

// Form constants
const FORM_TITLE = "Sign in to Hoglet Hub";
const EMAIL_LABEL = "Email Address";
const PASSWORD_LABEL = "Password";
const EMAIL_REQUIRED_ERROR = "Email is required";
const EMAIL_PATTERN_ERROR = "Please enter a valid email";
const PASSWORD_REQUIRED_ERROR = "Password is required";
const SIGNIN_BUTTON_TEXT = "Sign in";
const SIGNIN_LOADING_TEXT = "Signing in...";
const DEVELOPMENT_MODE_TEXT = "Development Mode";
const DEV_MODE_DESCRIPTION =
  "For development purposes, you can use any credentials to log in.";
const EMAIL_PATTERN = /\S+@\S+\.\S+/;

export default function LoginForm(): React.ReactElement {
  const { handleLogin, error, isLoading } = useLoginForm();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>();

  return (
    <div className="bg-white dark:bg-gray-800 py-8 px-6 shadow-md rounded-lg">
      <form
        onSubmit={handleSubmit(handleLogin)}
        className="space-y-6"
        aria-labelledby="login-heading"
      >
        <h2
          id="login-heading"
          className="text-2xl font-bold text-center text-gray-800 dark:text-white mb-8"
        >
          {FORM_TITLE}
        </h2>

        <div>
          <label
            id="email-label"
            htmlFor="email"
            className="block text-sm font-medium text-gray-700 dark:text-gray-200"
          >
            {EMAIL_LABEL}
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            aria-labelledby="email-label"
            aria-required="true"
            aria-invalid={!!errors.email}
            {...register("email", {
              required: EMAIL_REQUIRED_ERROR,
              pattern: {
                value: EMAIL_PATTERN,
                message: EMAIL_PATTERN_ERROR,
              },
            })}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
          />
          {errors.email && (
            <p className="mt-1 text-sm text-red-600" role="alert">
              {errors.email.message}
            </p>
          )}
        </div>

        <div>
          <label
            id="password-label"
            htmlFor="password"
            className="block text-sm font-medium text-gray-700 dark:text-gray-200"
          >
            {PASSWORD_LABEL}
          </label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            aria-labelledby="password-label"
            aria-required="true"
            aria-invalid={!!errors.password}
            {...register("password", {
              required: PASSWORD_REQUIRED_ERROR,
            })}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
          />
          {errors.password && (
            <p className="mt-1 text-sm text-red-600" role="alert">
              {errors.password.message}
            </p>
          )}
        </div>

        {error && (
          <div
            className="bg-red-100 text-red-700 p-3 rounded-md"
            role="alert"
            aria-live="assertive"
          >
            <p>{error}</p>
          </div>
        )}

        <div>
          <button
            type="submit"
            disabled={isLoading}
            aria-busy={isLoading}
            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:bg-blue-300"
          >
            {isLoading ? SIGNIN_LOADING_TEXT : SIGNIN_BUTTON_TEXT}
          </button>
        </div>
      </form>

      <div className="mt-6">
        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white text-gray-500">
              {DEVELOPMENT_MODE_TEXT}
            </span>
          </div>
        </div>
        <div className="mt-6 text-center text-sm text-gray-500">
          <p>{DEV_MODE_DESCRIPTION}</p>
        </div>
      </div>
    </div>
  );
}
