"use client";

import * as React from "react";
import { usePathname, useRouter } from "next/navigation";

import { useAuthStore } from "@/lib/store/auth";

/**
 * Gates the authenticated app shell. The access token lives in memory only, so
 * on a fresh load `SessionBootstrap` (in app/providers.tsx) first attempts a
 * silent refresh from the HttpOnly cookie and flips `initialized` when done.
 *
 * Until that resolves we show a calm loading state instead of rendering a page
 * that would immediately 401. Once resolved, an unauthenticated visitor is sent
 * to /login (preserving where they were headed via ?next=), which is the path
 * that was previously missing — landing on /dashboard with no session left the
 * user stuck on an error with no way to reach the login screen.
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const initialized = useAuthStore((s) => s.initialized);
  const authed = useAuthStore((s) => s.accessToken !== null);

  React.useEffect(() => {
    if (initialized && !authed) {
      const next = pathname && pathname !== "/" ? `?next=${encodeURIComponent(pathname)}` : "";
      router.replace(`/login${next}`);
    }
  }, [initialized, authed, pathname, router]);

  // Authenticated → render the app.
  if (authed) return <>{children}</>;

  // Still resolving the session, or about to redirect to /login.
  return (
    <div
      className="grid min-h-screen place-items-center bg-background"
      role="status"
      aria-live="polite"
    >
      <div className="flex flex-col items-center gap-3 text-muted-foreground">
        <div
          className="size-6 animate-spin rounded-full border-2 border-border border-t-primary"
          aria-hidden
        />
        <span className="text-sm">{initialized ? "Redirecting to sign in…" : "Loading…"}</span>
      </div>
    </div>
  );
}
