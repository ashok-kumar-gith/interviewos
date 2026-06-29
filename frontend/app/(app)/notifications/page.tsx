"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, CheckCheck, RefreshCw } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type Notification,
} from "@/lib/api/notifications";
import { NOTIFICATIONS_KEY } from "@/components/shell/notification-bell";
import { cn } from "@/lib/utils";

export default function NotificationsPage() {
  const queryClient = useQueryClient();

  const query = useQuery<Notification[], unknown>({
    queryKey: NOTIFICATIONS_KEY,
    queryFn: () => listNotifications(),
  });

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

  const notifications = query.data ?? [];
  const unreadCount = notifications.filter((n) => n.status === "unread").length;

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-h1">Notifications</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {unreadCount > 0 ? `${unreadCount} unread` : "You're all caught up."}
          </p>
        </div>
        {unreadCount > 0 && (
          <Button
            variant="outline"
            onClick={() => markAllMutation.mutate()}
            loading={markAllMutation.isPending}
          >
            <CheckCheck aria-hidden /> Mark all read
          </Button>
        )}
      </header>

      {query.isLoading ? (
        <div className="space-y-3" aria-busy>
          <span className="sr-only" role="status">
            Loading notifications
          </span>
          {[0, 1, 2, 3].map((i) => (
            <Card key={i} className="p-5">
              <Skeleton className="h-5 w-1/2" />
              <Skeleton className="mt-2 h-4 w-3/4" />
            </Card>
          ))}
        </div>
      ) : query.isError ? (
        <Alert variant="danger" title="Couldn't load notifications">
          Something went wrong.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => query.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      ) : notifications.length === 0 ? (
        <EmptyState
          icon={Bell}
          title="Nothing here yet"
          description="Reminders about your plan, revisions, and milestones will show up here."
        />
      ) : (
        <ul className="space-y-2">
          {notifications.map((n) => (
            <li key={n.id}>
              <Card
                className={cn(
                  "transition-colors",
                  n.status === "unread" && "border-primary/40",
                )}
              >
                <CardContent className="flex items-start gap-3 p-4">
                  <span
                    className={cn(
                      "mt-1.5 size-2 shrink-0 rounded-full",
                      n.status === "unread" ? "bg-primary" : "bg-transparent",
                    )}
                    aria-hidden
                  />
                  <div className="min-w-0 flex-1">
                    <p className="font-medium leading-snug">{n.title}</p>
                    {n.body && <p className="text-sm text-muted-foreground">{n.body}</p>}
                    {n.created_at && (
                      <p className="mt-1 text-xs text-muted-foreground">{formatDate(n.created_at)}</p>
                    )}
                  </div>
                  {n.status === "unread" && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => markReadMutation.mutate(n.id)}
                      loading={markReadMutation.isPending && markReadMutation.variables === n.id}
                    >
                      Mark read
                    </Button>
                  )}
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}
