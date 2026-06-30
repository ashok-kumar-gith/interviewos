"use client";

import * as React from "react";
import { Button } from "@/components/ui/button";
import { oauthStartUrl, type OAuthProvider } from "@/lib/api/auth";
import { cn } from "@/lib/utils";

/** Inline brand glyphs (lucide has no brand marks; keep self-contained SVGs). */
function GoogleIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden className="size-4">
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.27-4.74 3.27-8.1Z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.65l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A11 11 0 0 0 12 23Z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.11a6.6 6.6 0 0 1 0-4.22V7.05H2.18a11 11 0 0 0 0 9.9l3.66-2.84Z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1A11 11 0 0 0 2.18 7.05l3.66 2.84C6.71 7.3 9.14 5.38 12 5.38Z"
      />
    </svg>
  );
}

function GitHubIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden className="size-4 fill-current">
      <path d="M12 1.27a11 11 0 0 0-3.48 21.46c.55.1.75-.24.75-.53v-1.85c-3.06.67-3.7-1.47-3.7-1.47-.5-1.27-1.22-1.61-1.22-1.61-1-.68.08-.67.08-.67 1.1.08 1.68 1.13 1.68 1.13.98 1.68 2.57 1.2 3.2.92.1-.71.38-1.2.69-1.47-2.44-.28-5.01-1.22-5.01-5.43 0-1.2.43-2.18 1.13-2.95-.11-.28-.49-1.4.11-2.91 0 0 .92-.3 3.02 1.13a10.5 10.5 0 0 1 5.5 0c2.1-1.43 3.02-1.13 3.02-1.13.6 1.51.22 2.63.11 2.91.7.77 1.13 1.75 1.13 2.95 0 4.22-2.58 5.15-5.03 5.42.4.34.74 1 .74 2.02v3c0 .29.2.64.76.53A11 11 0 0 0 12 1.27Z" />
    </svg>
  );
}

export function OAuthButtons({ className }: { className?: string }) {
  const [busy, setBusy] = React.useState<OAuthProvider | null>(null);
  const [notice, setNotice] = React.useState<string | null>(null);

  // Probe the provider's /start endpoint instead of hard-navigating to it. A
  // configured provider responds with a redirect (302) to its auth page, which
  // we follow. An unconfigured one (the local default) responds 501, so we show
  // a friendly inline message instead of dumping the user on a raw error page.
  async function start(provider: OAuthProvider, label: string) {
    setNotice(null);
    setBusy(provider);
    try {
      const res = await fetch(oauthStartUrl(provider), {
        method: "GET",
        redirect: "manual",
        credentials: "include",
      });
      // Opaque redirect (type "opaqueredirect") or a 3xx means it's configured —
      // hand off to a full navigation so the browser follows the provider flow.
      if (res.type === "opaqueredirect" || (res.status >= 300 && res.status < 400)) {
        window.location.href = oauthStartUrl(provider);
        return;
      }
      if (res.status === 501) {
        setNotice(`${label} sign-in isn't set up yet — use email below to continue.`);
        return;
      }
      setNotice(`Couldn't start ${label} sign-in. Use email below to continue.`);
    } catch {
      setNotice(`Couldn't reach the server for ${label} sign-in. Use email below.`);
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className={cn("space-y-3", className)}>
      <div className="grid grid-cols-2 gap-3">
        <Button
          type="button"
          variant="outline"
          loading={busy === "google"}
          onClick={() => start("google", "Google")}
        >
          {busy !== "google" && <GoogleIcon />}
          Google
        </Button>
        <Button
          type="button"
          variant="outline"
          loading={busy === "github"}
          onClick={() => start("github", "GitHub")}
        >
          {busy !== "github" && <GitHubIcon />}
          GitHub
        </Button>
      </div>
      {notice && (
        <p role="status" className="text-center text-xs text-muted-foreground">
          {notice}
        </p>
      )}
    </div>
  );
}
