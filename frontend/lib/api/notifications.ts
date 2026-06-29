/**
 * Notifications API layer — list, mark-read, mark-all-read.
 * Shapes mirror the OpenAPI schemas: Notification, NotificationStatus.
 */

import { api } from "@/lib/api/client";

export type NotificationType =
  | "today_plan"
  | "revision_due"
  | "weekly_review"
  | "missed_goal"
  | "streak_reminder"
  | "readiness_milestone"
  | "mock_scheduled"
  | "system";

export type NotificationStatus = "unread" | "read" | "dismissed";

export interface Notification {
  id: string;
  type: NotificationType;
  channel?: string;
  status: NotificationStatus;
  title: string;
  body?: string | null;
  payload?: Record<string, unknown>;
  read_at?: string | null;
  created_at?: string;
}

/** GET /notifications — optionally filtered by status. */
export function listNotifications(status?: NotificationStatus): Promise<Notification[]> {
  return api
    .getList<Notification>("/notifications", {
      query: { page_size: 50, status },
    })
    .then((r) => r.data);
}

/** POST /notifications/{id}/read — mark a single notification read. */
export function markNotificationRead(id: string): Promise<Notification> {
  return api.post<Notification>(`/notifications/${id}/read`);
}

/** POST /notifications/read-all — mark all read. */
export function markAllNotificationsRead(): Promise<void> {
  return api.post<void>("/notifications/read-all");
}
