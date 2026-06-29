import type { ReactNode } from "react";
import { GraduationCap } from "lucide-react";
import { ThemeSync } from "@/components/shell/theme-sync";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-background px-4 py-12">
      <ThemeSync />

      {/* Ambient brand glow — calm, premium, dark-first. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 [mask-image:radial-gradient(60%_55%_at_50%_25%,black,transparent)]"
        style={{
          background:
            "radial-gradient(55% 45% at 50% 0%, hsl(var(--primary) / 0.18), transparent 70%)",
        }}
      />

      <div className="w-full max-w-[400px]">
        <div className="mb-8 flex flex-col items-center text-center">
          <div
            className="grid size-11 place-items-center rounded-xl bg-primary text-primary-foreground shadow-glow-primary"
            aria-hidden
          >
            <GraduationCap className="size-6" />
          </div>
          <span className="mt-3 text-h3 font-semibold tracking-tight">InterviewOS</span>
          <p className="mt-1 text-sm text-muted-foreground">
            Your interview prep, on autopilot.
          </p>
        </div>

        {children}

        <p className="mt-8 text-center text-xs text-muted-foreground">
          Calm, focused prep — one plan, every day.
        </p>
      </div>
    </div>
  );
}
