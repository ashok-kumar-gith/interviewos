import {
  BookOpen,
  ExternalLink,
  FileCode,
  FileText,
  GraduationCap,
  Newspaper,
  NotebookPen,
  PlayCircle,
  Terminal,
  type LucideIcon,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { DifficultyPill } from "@/components/ui/difficulty-pill";
import type { Resource } from "@/lib/api/content";
import type { ResourceType } from "@/lib/api/types";

const TYPE_ICON: Record<ResourceType, LucideIcon> = {
  book: BookOpen,
  video: PlayCircle,
  article: Newspaper,
  course: GraduationCap,
  github: FileCode,
  practice: Terminal,
  documentation: FileText,
  blog: NotebookPen,
  cheatsheet: FileText,
};

const TYPE_LABEL: Record<ResourceType, string> = {
  book: "Book",
  video: "Video",
  article: "Article",
  course: "Course",
  github: "GitHub",
  practice: "Practice",
  documentation: "Docs",
  blog: "Blog",
  cheatsheet: "Cheatsheet",
};

const PRIORITY_VARIANT = {
  critical: "danger",
  high: "warning",
  medium: "secondary",
  low: "outline",
} as const;

export function ResourceRow({ resource }: { resource: Resource }) {
  const Icon = TYPE_ICON[resource.type] ?? FileText;
  const body = (
    <div className="flex items-start gap-3 rounded-md border border-border bg-card p-3 transition-colors hover:bg-muted/40">
      <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
        <Icon className="size-4" aria-hidden />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-medium">{resource.title}</p>
          {resource.url && (
            <ExternalLink className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          )}
        </div>
        {resource.description && (
          <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
            {resource.description}
          </p>
        )}
        <div className="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <Badge variant="outline" size="sm">
            {TYPE_LABEL[resource.type] ?? resource.type}
          </Badge>
          {resource.author && <span className="truncate">{resource.author}</span>}
          {resource.provider && !resource.author && (
            <span className="truncate">{resource.provider}</span>
          )}
          {resource.estimated_minutes != null && (
            <span className="tabular-nums">{resource.estimated_minutes}m</span>
          )}
          {resource.difficulty && <DifficultyPill difficulty={resource.difficulty} />}
          {resource.priority && resource.priority !== "medium" && (
            <Badge variant={PRIORITY_VARIANT[resource.priority]} size="sm" className="capitalize">
              {resource.priority}
            </Badge>
          )}
          {resource.is_free === false && (
            <Badge variant="warning" size="sm">
              Paid
            </Badge>
          )}
        </div>
      </div>
    </div>
  );

  if (resource.url) {
    return (
      <a
        href={resource.url}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={`${resource.title} (opens in a new tab)`}
        className="block rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {body}
      </a>
    );
  }
  return body;
}
