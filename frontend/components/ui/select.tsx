import * as React from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement>;

/**
 * Native `<select>` styled to match the design system. Native semantics give us
 * full keyboard support, typeahead, and AA accessibility for free (no Radix
 * dependency required). Label/error wiring is handled by the caller via
 * `id` + `aria-describedby` (see SelectField).
 */
const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, children, ...props }, ref) => {
    return (
      <div className="relative">
        <select
          ref={ref}
          className={cn(
            "flex h-9 w-full appearance-none rounded-sm border border-input bg-background pl-3 pr-9 py-1 text-sm",
            "text-foreground",
            "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring " +
              "focus-visible:ring-offset-2 focus-visible:ring-offset-background",
            "disabled:cursor-not-allowed disabled:opacity-50",
            "aria-[invalid=true]:border-danger aria-[invalid=true]:focus-visible:ring-danger",
            // Empty (placeholder) value reads as muted, like an input placeholder.
            "[&:has(option[value='']:checked)]:text-muted-foreground",
            className,
          )}
          {...props}
        >
          {children}
        </select>
        <ChevronDown
          aria-hidden
          className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
      </div>
    );
  },
);
Select.displayName = "Select";

export { Select };
