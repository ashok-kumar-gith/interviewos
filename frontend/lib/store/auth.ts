import { create } from "zustand";
import type { User } from "@/lib/api/auth";

/**
 * Auth session state (ARCHITECTURE §5).
 *
 * The short-lived access token lives **in memory only** (never localStorage),
 * mitigating XSS token theft. The refresh token is held by the backend in an
 * HttpOnly cookie. This store is the single source of truth for the access
 * token; `lib/api/client.ts` reads it via the registered token provider.
 */
interface AuthState {
  accessToken: string | null;
  user: User | null;
  /** True once an initial session check has resolved (success or failure). */
  initialized: boolean;
  setSession: (session: { accessToken: string; user: User }) => void;
  setUser: (user: User) => void;
  clearSession: () => void;
  setInitialized: (initialized: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  initialized: false,
  setSession: ({ accessToken, user }) => set({ accessToken, user }),
  setUser: (user) => set({ user }),
  clearSession: () => set({ accessToken: null, user: null }),
  setInitialized: (initialized) => set({ initialized }),
}));

/** Read the access token outside React (used by the API client token provider). */
export function getAccessToken(): string | null {
  return useAuthStore.getState().accessToken;
}
