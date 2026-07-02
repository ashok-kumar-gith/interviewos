"use client";

import * as React from "react";
import { X } from "lucide-react";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

export interface TagInputProps {
  /** Visible field label, associated via the input's id. */
  label: string;
  /** Required for label/error wiring. */
  id: string;
  /** Current tags (controlled). */
  value: string[];
  /** Called with the next tag array on add/remove. */
  onChange: (value: string[]) => void;
  placeholder?: string;
  /** Inline validation message rendered below the control. */
  error?: string;
  /** Optional helper text shown when there is no error. */
  hint?: string;
  disabled?: boolean;
  /** Lowercase + strip spaces (used for slug-style tags like pattern_slugs). */
  slugify?: boolean;
  className?: string;
}

/**
 * A token/tag input in the token design style: type a value and press Enter or
 * comma to add it; Backspace on an empty field removes the last tag; each tag
 * has a remove button. Fully keyboard-accessible and screen-reader labelled.
 */
export function TagInput({
  label,
  id,
  value,
  onChange,
  placeholder,
  error,
  hint,
  disabled,
  slugify,
  className,
}: TagInputProps) {
  const [draft, setDraft] = React.useState("");
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;

  const normalize = React.useCallback(
    (raw: string): string => {
      const t = raw.trim();
      return slugify ? t.toLowerCase().replace(/\s+/g, "-") : t;
    },
    [slugify],
  );

  const addTag = React.useCallback(
    (raw: string) => {
      const tag = normalize(raw);
      if (!tag) return;
      if (value.includes(tag)) {
        setDraft("");
        return;
      }
      onChange([...value, tag]);
      setDraft("");
    },
    [normalize, onChange, value],
  );

  const removeAt = React.useCallback(
    (index: number) => {
      onChange(value.filter((_, i) => i !== index));
    },
    [onChange, value],
  );

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      addTag(draft);
    } else if (e.key === "Backspace" && draft === "" && value.length > 0) {
      e.preventDefault();
      removeAt(value.length - 1);
    }
  }

  return (
    <div className={cn("space-y-1.5", className)}>
      <div className="flex items-baseline justify-between gap-2">
        <Label htmlFor={id}>{label}</Label>
        <span className="text-xs text-muted-foreground">Optional</span>
      </div>
      <div
        className={cn(
          "flex min-h-9 w-full flex-wrap items-center gap-1.5 rounded-sm border border-input bg-background px-2 py-1.5 text-sm",
          "focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2 focus-within:ring-offset-background",
          error && "border-danger focus-within:ring-danger",
          disabled && "cursor-not-allowed opacity-50",
        )}
      >
        {value.map((tag, i) => (
          <span
            key={`${tag}-${i}`}
            className="inline-flex items-center gap-1 rounded-sm border border-border bg-muted px-1.5 py-0.5 text-xs font-medium"
          >
            {tag}
            <button
              type="button"
              onClick={() => removeAt(i)}
              disabled={disabled}
              aria-label={`Remove ${tag}`}
              className="rounded-sm text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              <X className="size-3" aria-hidden />
            </button>
          </span>
        ))}
        <input
          id={id}
          type="text"
          value={draft}
          disabled={disabled}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={onKeyDown}
          onBlur={() => addTag(draft)}
          placeholder={value.length === 0 ? placeholder : undefined}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errorId : hintId}
          className="min-w-[8ch] flex-1 bg-transparent px-1 py-0.5 text-foreground placeholder:text-muted-foreground focus:outline-none"
        />
      </div>
      {error ? (
        <p id={errorId} className="text-xs font-medium text-danger" role="alert">
          {error}
        </p>
      ) : hint ? (
        <p id={hintId} className="text-xs text-muted-foreground">
          {hint}
        </p>
      ) : null}
    </div>
  );
}
