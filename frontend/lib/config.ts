/**
 * App-wide configuration derived from the environment.
 *
 * Values here are read at build time (Next.js inlines `NEXT_PUBLIC_*` vars), so
 * changing them requires a redeploy — that's fine for pagination sizing.
 */

/**
 * Number of items shown per page in paginated catalog lists (problems,
 * resources, backend-engineering, and the ordered problem list).
 *
 * Configurable via `NEXT_PUBLIC_PAGE_SIZE`; defaults to 10. Falls back to 10 for
 * any non-positive or non-numeric value.
 */
export const PAGE_SIZE: number = (() => {
  const raw = Number(process.env.NEXT_PUBLIC_PAGE_SIZE);
  return Number.isInteger(raw) && raw > 0 ? raw : 10;
})();

/**
 * Whether the in-browser code runner is available. The backend executor
 * (`/code/run`) is disabled on the public deployment for security (it runs
 * arbitrary code with server privileges), so the UI must not offer a Run button
 * that would 404. Enabled only when `NEXT_PUBLIC_CODE_RUNNER_ENABLED=true`
 * (set it in local dev, where the backend runs the executor).
 */
export const CODE_RUNNER_ENABLED: boolean =
  process.env.NEXT_PUBLIC_CODE_RUNNER_ENABLED === "true";
