"use client";

import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setAccessTokenProvider, setRefreshHandler } from "@/lib/api/client";
import { getAccessToken, useAuthStore } from "@/lib/store/auth";
import { refresh as refreshSession } from "@/lib/api/auth";

// Wire the API client to read the in-memory access token from the auth store.
// Registered at module load so it is set before any request fires.
setAccessTokenProvider(getAccessToken);

// Wire transparent re-auth: when a request 401s, the client calls this to mint a
// fresh access token from the HttpOnly refresh cookie and store it, then retries.
// Returns false when no valid session exists (e.g. logged out / cookie expired).
setRefreshHandler(async () => {
  try {
    const tokens = await refreshSession();
    useAuthStore.getState().setSession({ accessToken: tokens.access_token, user: tokens.user });
    return true;
  } catch {
    useAuthStore.getState().clearSession();
    return false;
  }
});

/**
 * Restores the session on app load. The access token is in memory only, so after
 * a reload it is gone; we attempt a single refresh (via the HttpOnly cookie) to
 * re-mint it before marking the auth store initialized. This is what keeps
 * Dashboard/Today/Roadmap/Revision/Resume/Coach working across reloads and after
 * the short-lived access token expires.
 */
function SessionBootstrap(): null {
  const initialized = useAuthStore((s) => s.initialized);
  const hasToken = useAuthStore((s) => s.accessToken !== null);

  React.useEffect(() => {
    if (initialized || hasToken) return;
    let cancelled = false;
    void (async () => {
      try {
        const tokens = await refreshSession();
        if (!cancelled) {
          useAuthStore.getState().setSession({ accessToken: tokens.access_token, user: tokens.user });
        }
      } catch {
        if (!cancelled) useAuthStore.getState().clearSession();
      } finally {
        if (!cancelled) useAuthStore.getState().setInitialized(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [initialized, hasToken]);

  return null;
}

function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60_000, // 1 min — server state stays fresh briefly
        gcTime: 5 * 60_000,
        retry: 1,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: 0,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

function getQueryClient(): QueryClient {
  if (typeof window === "undefined") {
    // Server: always make a fresh client per request.
    return makeQueryClient();
  }
  // Browser: reuse a single client across renders.
  if (!browserQueryClient) browserQueryClient = makeQueryClient();
  return browserQueryClient;
}

export function Providers({ children }: { children: React.ReactNode }) {
  const queryClient = getQueryClient();
  return (
    <QueryClientProvider client={queryClient}>
      <SessionBootstrap />
      {children}
    </QueryClientProvider>
  );
}
