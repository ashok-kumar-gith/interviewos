import * as React from "react";
import { Label } from "@/components/ui/label";
import { Select, type SelectProps } from "@/components/ui/select";
import { cn } from "@/lib/utils";

export interface SelectFieldProps extends SelectProps {
  /** Visible field label, associated via `htmlFor`/`id`. */
  label: string;
  /** Required for label/error wiring. */
  id: string;
  /** Inline validation message rendered below the select. */
  error?: string;
  /** Optional helper text shown when there is no error. */
  hint?: string;
  /** Wrapper className (the select keeps its own). */
  containerClassName?: string;
}

/**
 * Label + Select + error/hint message, wired for accessibility — the sibling of
 * FormField but for a native `<select>`.
 */
const SelectField = React.forwardRef<HTMLSelectElement, SelectFieldProps>(
  ({ label, id, error, hint, containerClassName, required, children, ...selectProps }, ref) => {
    const hintId = hint ? `${id}-hint` : undefined;
    const errorId = error ? `${id}-error` : undefined;
    const describedBy = error ? errorId : hintId;

    return (
      <div className={cn("space-y-1.5", containerClassName)}>
        <div className="flex items-baseline justify-between gap-2">
          <Label htmlFor={id}>{label}</Label>
          {!required && <span className="text-xs text-muted-foreground">Optional</span>}
        </div>
        <Select
          ref={ref}
          id={id}
          required={required}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          {...selectProps}
        >
          {children}
        </Select>
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
  },
);
SelectField.displayName = "SelectField";

export { SelectField };
