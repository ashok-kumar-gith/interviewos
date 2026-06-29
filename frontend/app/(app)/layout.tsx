import type { ReactNode } from "react";
import { Sidebar } from "@/components/shell/sidebar";
import { Topbar } from "@/components/shell/topbar";
import { ThemeSync } from "@/components/shell/theme-sync";
import { KeyboardShortcuts } from "@/components/shell/keyboard-shortcuts";
import { NotificationGenerator } from "@/components/shell/notification-generator";
import { CommandPalette } from "@/components/command-palette/command-palette";
import { AuthGuard } from "@/components/shell/auth-guard";

export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <AuthGuard>
      <div className="flex min-h-screen flex-col">
        <ThemeSync />
        <KeyboardShortcuts />
        <NotificationGenerator />
        <CommandPalette />
        <Topbar />
        <div className="flex flex-1 overflow-hidden">
          <Sidebar />
          <main className="flex-1 overflow-y-auto bg-background">
            <div className="mx-auto w-full max-w-[1200px] px-4 py-8 md:px-6">{children}</div>
          </main>
        </div>
      </div>
    </AuthGuard>
  );
}
