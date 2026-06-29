"use client";

import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setAccessTokenProvider } from "@/lib/api/client";
import { getAccessToken } from "@/lib/store/auth";

// Wire the API client to read the in-memory access token from the auth store.
// Registered at module load so it is set before any request fires.
setAccessTokenProvider(getAccessToken);

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
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
