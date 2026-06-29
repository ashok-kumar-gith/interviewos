"use client";

import * as React from "react";
import { useQueryClient } from "@tanstack/react-query";

import { generateNotifications } from "@/lib/api/notifications";
import { NOTIFICATIONS_KEY } from "@/components/shell/notification-bell";

/**
 * Fires the digest-notification generation once per app session when the app
 * shell mounts, then refreshes the notifications query so the bell reflects the
 * freshly generated digests. Fire-and-forget: any error is swallowed so it can
 * never break the shell. Renders nothing.
 */
export function NotificationGenerator() {
  const queryClient = useQueryClient();
  const ranRef = React.useRef(false);

  React.useEffect(() => {
    if (ranRef.current) return;
    ranRef.current = true;

    let cancelled = false;
    void generateNotifications()
      .then(() => {
        if (!cancelled) {
          void queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY });
        }
      })
      .catch(() => {
        // Intentionally ignore — notification generation is best-effort.
      });

    return () => {
      cancelled = true;
    };
  }, [queryClient]);

  return null;
}
