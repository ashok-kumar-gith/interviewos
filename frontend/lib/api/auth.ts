/**
 * Auth API layer — typed wrappers over the InterviewOS auth endpoints.
 * Request/response shapes mirror the OpenAPI schemas (docs/openapi.yaml):
 * RegisterRequest, LoginRequest, RefreshRequest, ForgotPasswordRequest,
 * ResetPasswordRequest, AuthTokensResponse, User.
 */

import { api, API_BASE } from "@/lib/api/client";

export type UserRole = "user" | "admin";
export type UserStatus = "active" | "suspended" | "deleted";

export interface User {
  id: string;
  email: string;
  email_verified?: boolean;
  full_name?: string | null;
  avatar_url?: string | null;
  role: UserRole;
  status: UserStatus;
  last_login_at?: string | null;
  created_at: string;
  updated_at?: string;
}

export interface AuthTokensResponse {
  access_token: string;
  /** Also set as an HttpOnly cookie by the backend. */
  refresh_token?: string;
  token_type: "Bearer";
  /** Access token TTL in seconds. */
  expires_in: number;
  user: User;
}

export interface RegisterRequest {
  email: string;
  password: string;
  full_name?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RefreshRequest {
  refresh_token?: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  token: string;
  password: string;
}

export type OAuthProvider = "google" | "github";

export function register(body: RegisterRequest): Promise<AuthTokensResponse> {
  return api.post<AuthTokensResponse>("/auth/register", body);
}

export function login(body: LoginRequest): Promise<AuthTokensResponse> {
  return api.post<AuthTokensResponse>("/auth/login", body);
}

/** Rotate the refresh token (sent via HttpOnly cookie) and mint a new access token. */
export function refresh(body: RefreshRequest = {}): Promise<AuthTokensResponse> {
  return api.post<AuthTokensResponse>("/auth/refresh", body);
}

export function logout(): Promise<void> {
  return api.post<void>("/auth/logout");
}

export function forgotPassword(body: ForgotPasswordRequest): Promise<void> {
  return api.post<void>("/auth/forgot-password", body);
}

export function resetPassword(body: ResetPasswordRequest): Promise<void> {
  return api.post<void>("/auth/reset-password", body);
}

/** The authenticated user (requires a valid access token). */
export function getMe(): Promise<User> {
  return api.get<User>("/auth/me");
}

/**
 * Absolute URL to begin a backend OAuth redirect flow. The backend sets the
 * refresh cookie and returns the access token to the SPA callback route.
 */
export function oauthStartUrl(provider: OAuthProvider): string {
  return `${API_BASE}/auth/oauth/${provider}/start`;
}
