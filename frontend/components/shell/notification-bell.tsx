"use client";

import * as React from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, CheckCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover } from "@/components/ui/popover";
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type Notification,
} from "@/lib/api/notifications";
import { cn } from "@/lib/utils";

export const NOTIFICATIONS_KEY = ["notifications"] as const;

/**
 * Top-bar notification bell: unread count, dropdown list, mark-read /
 * mark-all-read. Wired to the notifications API via React Query.
 */
export function NotificationBell() {
  const queryClient = useQueryClient();
  const [open, setOpen] = React.useState(false);

  const query = useQuery<Notification[], unknown>({
    queryKey: NOTIFICATIONS_KEY,
    queryFn: () => listNotifications(),
    // Soft-fail: the bell should never break the shell.
    retry: 1,
  });

  const notifications = query.data ?? [];
  const unread = notifications.filter((n) => n.status === "unread");

  const markReadMutation = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: NOTIFICATIONS_KEY });
      const previous = queryClient.getQueryData<Notification[]>(NOTIFICATIONS_KEY);
      queryClient.setQueryData<Notification[]>(NOTIFICATIONS_KEY, (old) =>
        old?.map((n) => (n.id === id ? { ...n, status: "read" } : n)),
      );
      return { previous };
    },
    onError: (_e, _id, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(NOTIFICATIONS_KEY, ctx.previous);
    },
    onSettled: () => void queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY }),
  });

  const markAllMutation = useMutation({
    mutationFn: () => markAllNotificationsRead(),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: NOTIFICATIONS_KEY });
      const previous = queryClient.getQueryData<Notification[]>(NOTIFICATIONS_KEY);
      queryClient.setQueryData<Notification[]>(NOTIFICATIONS_KEY, (old) =>
        old?.map((n) => (n.status === "unread" ? { ...n, status: "read" } : n)),
      );
      return { previous };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(NOTIFICATIONS_KEY, ctx.previous);
    },
    onSettled: () => void queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY }),
  });

  return (
    <Popover
      open={open}
      onOpenChange={setOpen}
      align="end"
      ariaLabel="Notifications"
      className="w-80 max-w-[calc(100vw-2rem)]"
      trigger={
        <Button
          variant="ghost"
          size="icon"
          className="relative size-8"
          aria-label={
            unread.length > 0 ? `Notifications, ${unread.length} unread` : "Notifications"
          }
          aria-haspopup="dialog"
          aria-expanded={open}
          onClick={() => setOpen((o) => !o)}
        >
          <Bell className="size-4" />
          {unread.length > 0 && (
            <span
              className="absolute -right-0.5 -top-0.5 grid min-w-4 place-items-center rounded-full bg-danger px-1 text-2xs font-semibold text-danger-foreground"
              aria-hidden
            >
              {unread.length > 9 ? "9+" : unread.length}
            </span>
          )}
        </Button>
      }
    >
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-sm font-medium">Notifications</span>
        {unread.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => markAllMutation.mutate()}
            loading={markAllMutation.isPending}
          >
            <CheckCheck aria-hidden /> Mark all read
          </Button>
        )}
      </div>

      <div className="max-h-96 overflow-y-auto">
        {query.isLoading ? (
          <p className="px-3 py-6 text-center text-sm text-muted-foreground" role="status">
            Loading…
          </p>
        ) : query.isError ? (
          <p className="px-3 py-6 text-center text-sm text-muted-foreground">
            Couldn&apos;t load notifications.
          </p>
        ) : notifications.length === 0 ? (
          <p className="px-3 py-8 text-center text-sm text-muted-foreground">
            You&apos;re all caught up.
          </p>
        ) : (
          <ul>
            {notifications.slice(0, 12).map((n) => (
              <li key={n.id}>
                <button
                  type="button"
                  onClick={() => {
                    if (n.status === "unread") markReadMutation.mutate(n.id);
                  }}
                  className={cn(
                    "flex w-full items-start gap-2 px-3 py-2.5 text-left transition-colors hover:bg-muted/60",
                    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring",
                  )}
                >
                  <span
                    className={cn(
                      "mt-1.5 size-2 shrink-0 rounded-full",
                      n.status === "unread" ? "bg-primary" : "bg-transparent",
                    )}
                    aria-hidden
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block text-sm font-medium leading-snug">{n.title}</span>
                    {n.body && (
                      <span className="block truncate text-xs text-muted-foreground">
                        {n.body}
                      </span>
                    )}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="border-t border-border px-3 py-2">
        <Link
          href="/notifications"
          onClick={() => setOpen(false)}
          className="text-sm text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          View all
        </Link>
      </div>
    </Popover>
  );
}
