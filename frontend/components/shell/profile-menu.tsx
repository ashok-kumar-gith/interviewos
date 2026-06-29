"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { LogOut, Settings, User } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover } from "@/components/ui/popover";
import { logout as logoutApi } from "@/lib/api/auth";
import { useAuthStore } from "@/lib/store/auth";

/**
 * Topbar profile menu with a working Log out action. Previously the profile
 * button was inert (no onClick), leaving users with no way to sign out.
 *
 * Logout revokes the refresh token server-side (POST /auth/logout clears the
 * HttpOnly cookie), clears the in-memory access token, and routes to /login.
 * It is resilient: even if the API call fails (e.g. already-expired session),
 * the local session is cleared and the user is sent to /login regardless.
 */
export function ProfileMenu() {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const user = useAuthStore((s) => s.user);
  const clearSession = useAuthStore((s) => s.clearSession);

  async function handleLogout() {
    setBusy(true);
    try {
      await logoutApi();
    } catch {
      // Ignore — we clear the local session and redirect regardless.
    } finally {
      clearSession();
      setOpen(false);
      router.replace("/login");
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={setOpen}
      ariaLabel="Profile menu"
      className="w-56 p-1.5"
      trigger={
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          aria-label="Profile menu"
          aria-haspopup="menu"
          aria-expanded={open}
          onClick={() => setOpen((o) => !o)}
        >
          <User className="size-4" />
        </Button>
      }
    >
      <div className="px-2 py-1.5">
        <p className="truncate text-sm font-medium">{user?.full_name || "Your account"}</p>
        {user?.email && (
          <p className="truncate text-xs text-muted-foreground">{user.email}</p>
        )}
      </div>
      <div className="my-1 h-px bg-border" />
      <button
        type="button"
        role="menuitem"
        onClick={() => {
          setOpen(false);
          router.push("/settings");
        }}
        className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <Settings className="size-4" aria-hidden />
        Settings
      </button>
      <button
        type="button"
        role="menuitem"
        onClick={handleLogout}
        disabled={busy}
        className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm text-danger transition-colors hover:bg-danger/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-60"
      >
        <LogOut className="size-4" aria-hidden />
        {busy ? "Signing out…" : "Log out"}
      </button>
    </Popover>
  );
}
