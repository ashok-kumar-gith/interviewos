"use client";

import { useAuthStore } from "@/lib/store/auth";

/**
 * True when the current session belongs to an admin (JWT `user.role === "admin"`).
 * Drives all authoring affordances — the admin nav entry, the "New content"
 * button, and the per-item edit/delete actions. Non-admins get `false`, so the
 * controls never render for them (defence in depth: the backend also enforces).
 */
export function useIsAdmin(): boolean {
  return useAuthStore((s) => s.user?.role === "admin");
}
