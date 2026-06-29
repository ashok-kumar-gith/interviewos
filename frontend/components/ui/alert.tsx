import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { CheckCircle2, Info, OctagonAlert, TriangleAlert, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

const alertVariants = cva(
  "flex items-start gap-2.5 rounded-md border px-3.5 py-3 text-sm [&_svg]:size-4 [&_svg]:shrink-0 [&_svg]:mt-0.5",
  {
    variants: {
      variant: {
        default: "border-border bg-muted/40 text-foreground [&_svg]:text-muted-foreground",
        info: "border-info/30 bg-info/10 text-foreground [&_svg]:text-info",
        success: "border-success/30 bg-success/10 text-foreground [&_svg]:text-success",
        warning: "border-warning/30 bg-warning/10 text-foreground [&_svg]:text-warning",
        danger: "border-danger/30 bg-danger/10 text-foreground [&_svg]:text-danger",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

const ICONS: Record<NonNullable<VariantProps<typeof alertVariants>["variant"]>, LucideIcon> = {
  default: Info,
  info: Info,
  success: CheckCircle2,
  warning: TriangleAlert,
  danger: OctagonAlert,
};

export interface AlertProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof alertVariants> {
  /** Set false to suppress the leading status icon. */
  withIcon?: boolean;
  title?: string;
}

const Alert = React.forwardRef<HTMLDivElement, AlertProps>(
  ({ className, variant = "default", withIcon = true, title, children, ...props }, ref) => {
    const Icon = ICONS[variant ?? "default"];
    const isError = variant === "danger";
    return (
      <div
        ref={ref}
        role={isError ? "alert" : "status"}
        className={cn(alertVariants({ variant }), className)}
        {...props}
      >
        {withIcon && <Icon aria-hidden />}
        <div className="min-w-0 space-y-0.5">
          {title && <p className="font-medium leading-tight">{title}</p>}
          {children && <div className="text-muted-foreground [&_a]:text-primary">{children}</div>}
        </div>
      </div>
    );
  },
);
Alert.displayName = "Alert";

export { Alert, alertVariants };
