import { ApiError } from "@/lib/api/client";

/**
 * Turn any thrown error into blameless, actionable copy (DESIGN-SYSTEM §11).
 * Never surface raw codes or stack traces to the user.
 */
export function authErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    switch (error.code) {
      case "INVALID_CREDENTIALS":
      case "UNAUTHENTICATED":
        return "That email or password doesn't match. Try again.";
      case "CONFLICT":
        return "An account with this email already exists. Try signing in.";
      case "RATE_LIMITED":
        return "Too many attempts. Wait a moment and try again.";
      case "VALIDATION_ERROR":
        return error.details?.[0]?.message ?? "Please check the form and try again.";
      case "BAD_REQUEST":
        return error.message || "This reset link is invalid or has expired.";
      default:
        return error.status === 0
          ? "Couldn't reach the server. Check your connection and retry."
          : error.message || fallback;
    }
  }
  return fallback;
}
