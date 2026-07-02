"use client";

import * as React from "react";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";

/**
 * Turn an API error into a human message + per-field detail map. Handles the
 * two cases the authoring forms care about specially: 403 (not an admin) and
 * 422 (validation errors, which carry a `details` array of {field, message}).
 */
export function describeApiError(error: unknown): {
  message: string;
  fieldErrors: Record<string, string>;
} {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return {
        message: "You don't have permission to author content. Admin access is required.",
        fieldErrors: {},
      };
    }
    const fieldErrors: Record<string, string> = {};
    for (const d of error.details ?? []) {
      if (d.field && !fieldErrors[d.field]) fieldErrors[d.field] = d.message;
    }
    if (error.status === 422 || error.code === "VALIDATION_ERROR") {
      return {
        message: error.message || "Some fields need attention. Check the highlighted inputs.",
        fieldErrors,
      };
    }
    return { message: error.message || "Something went wrong. Try again.", fieldErrors };
  }
  return { message: "Something went wrong. Try again.", fieldErrors: {} };
}

export interface FormShellProps {
  onSubmit: (e: React.FormEvent<HTMLFormElement>) => void;
  submitting: boolean;
  submitLabel: string;
  /** Top-level error banner text (from describeApiError). */
  error?: string | null;
  onCancel?: () => void;
  children: React.ReactNode;
}

/** Shared chrome for an authoring form: error banner, fields, action row. */
export function FormShell({
  onSubmit,
  submitting,
  submitLabel,
  error,
  onCancel,
  children,
}: FormShellProps) {
  return (
    <form onSubmit={onSubmit} noValidate className="space-y-5">
      {error && <Alert variant="danger">{error}</Alert>}
      {children}
      <div className="flex items-center justify-end gap-2 pt-1">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel} disabled={submitting}>
            Cancel
          </Button>
        )}
        <Button type="submit" loading={submitting}>
          {submitLabel}
        </Button>
      </div>
    </form>
  );
}
