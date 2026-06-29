"use client";

import { useEffect } from "react";
import { useUiStore } from "@/lib/store/ui";

/**
 * Keeps the `<html>` element's `dark`/`light` class in sync with the persisted
 * UI store. Renders nothing. The root layout ships `class="dark"` for a correct
 * first paint (no flash); this reconciles once the persisted store hydrates.
 */
export function ThemeSync() {
  const theme = useUiStore((s) => s.theme);

  useEffect(() => {
    const root = document.documentElement;
    root.classList.toggle("dark", theme === "dark");
    root.classList.toggle("light", theme === "light");
  }, [theme]);

  return null;
}
