import type { ReactNode } from "react";
import { GraduationCap } from "lucide-react";
import { ThemeSync } from "@/components/shell/theme-sync";

/**
 * Centered, focused onboarding shell (Wizard / Intake page template,
 * DESIGN-SYSTEM §5.4) — no app chrome, calm ambient brand glow, dark-first.
 */
export default function OnboardingLayout({ children }: { children: ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-background px-4 py-12">
      <ThemeSync />

      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 [mask-image:radial-gradient(60%_55%_at_50%_15%,black,transparent)]"
        style={{
          background:
            "radial-gradient(55% 45% at 50% 0%, hsl(var(--primary) / 0.16), transparent 70%)",
        }}
      />

      <div className="w-full max-w-[560px]">
        <div className="mb-8 flex flex-col items-center text-center">
          <div
            className="grid size-11 place-items-center rounded-xl bg-primary text-primary-foreground shadow-glow-primary"
            aria-hidden
          >
            <GraduationCap className="size-6" />
          </div>
          <span className="mt-3 text-h3 font-semibold tracking-tight">InterviewOS</span>
        </div>

        {children}
      </div>
    </div>
  );
}
