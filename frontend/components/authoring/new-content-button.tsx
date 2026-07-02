"use client";

import Link from "next/link";
import { Plus } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { useIsAdmin } from "@/lib/store/admin";
import type { ContentType } from "@/components/authoring/shared";

/**
 * Admin-only "New …" button linking into the authoring page pre-selected to a
 * content type. Renders nothing for non-admins.
 */
export function NewContentButton({ type, label }: { type: ContentType; label: string }) {
  const isAdmin = useIsAdmin();
  if (!isAdmin) return null;
  return (
    <Link
      href={`/admin/content?type=${type}`}
      className={buttonVariants({ variant: "primary", size: "sm" })}
    >
      <Plus className="size-4" aria-hidden />
      {label}
    </Link>
  );
}
