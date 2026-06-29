import * as React from "react";
import { Label } from "@/components/ui/label";
import { Input, type InputProps } from "@/components/ui/input";
import { cn } from "@/lib/utils";

export interface FormFieldProps extends InputProps {
  /** Visible field label, associated via `htmlFor`/`id`. */
  label: string;
  /** Required for label/error wiring. */
  id: string;
  /** Inline validation message rendered below the input. */
  error?: string;
  /** Optional helper text shown when there is no error. */
  hint?: string;
}

/**
 * Label + Input + error/hint message, wired for accessibility:
 * `aria-invalid` + `aria-describedby` point the input at its message.
 */
const FormField = React.forwardRef<HTMLInputElement, FormFieldProps>(
  ({ label, id, error, hint, className, required, ...inputProps }, ref) => {
    const hintId = hint ? `${id}-hint` : undefined;
    const errorId = error ? `${id}-error` : undefined;
    const describedBy = error ? errorId : hintId;

    return (
      <div className={cn("space-y-1.5", className)}>
        <div className="flex items-baseline justify-between gap-2">
          <Label htmlFor={id}>{label}</Label>
          {!required && <span className="text-xs text-muted-foreground">Optional</span>}
        </div>
        <Input
          ref={ref}
          id={id}
          required={required}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          {...inputProps}
        />
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
FormField.displayName = "FormField";

export { FormField };
