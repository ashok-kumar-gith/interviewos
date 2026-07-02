import * as React from "react";
import { TextareaField, type TextareaFieldProps } from "@/components/ui/textarea-field";
import { cn } from "@/lib/utils";

/**
 * A TextareaField preset for authoring Markdown: monospace font, taller default,
 * and a standing hint that the content is rendered as Markdown. Used for all the
 * long-form `*_md` authoring fields (prompt_summary, approach_md, sections, …).
 */
const MarkdownTextarea = React.forwardRef<HTMLTextAreaElement, TextareaFieldProps>(
  ({ rows = 6, className, hint = "Markdown supported.", ...props }, ref) => {
    return (
      <TextareaField
        ref={ref}
        rows={rows}
        hint={props.error ? undefined : hint}
        className={cn("font-mono text-xs leading-relaxed", className)}
        {...props}
      />
    );
  },
);
MarkdownTextarea.displayName = "MarkdownTextarea";

export { MarkdownTextarea };
