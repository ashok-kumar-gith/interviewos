import type { Config } from "tailwindcss";
import animate from "tailwindcss-animate";

const config: Config = {
  darkMode: "class",
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./features/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        surface: { DEFAULT: "hsl(var(--surface))", foreground: "hsl(var(--surface-foreground))" },
        card: { DEFAULT: "hsl(var(--card))", foreground: "hsl(var(--card-foreground))" },
        popover: { DEFAULT: "hsl(var(--popover))", foreground: "hsl(var(--popover-foreground))" },
        muted: { DEFAULT: "hsl(var(--muted))", foreground: "hsl(var(--muted-foreground))" },
        primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" },
        secondary: { DEFAULT: "hsl(var(--secondary))", foreground: "hsl(var(--secondary-foreground))" },
        accent: { DEFAULT: "hsl(var(--accent))", foreground: "hsl(var(--accent-foreground))" },
        success: { DEFAULT: "hsl(var(--success))", foreground: "hsl(var(--success-foreground))" },
        warning: { DEFAULT: "hsl(var(--warning))", foreground: "hsl(var(--warning-foreground))" },
        danger: { DEFAULT: "hsl(var(--danger))", foreground: "hsl(var(--danger-foreground))" },
        info: { DEFAULT: "hsl(var(--info))", foreground: "hsl(var(--info-foreground))" },
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        pillar: {
          dsa: "hsl(var(--pillar-dsa))",
          "system-design": "hsl(var(--pillar-system-design))",
          lld: "hsl(var(--pillar-lld))",
          backend: "hsl(var(--pillar-backend))",
          behavioral: "hsl(var(--pillar-behavioral))",
          resume: "hsl(var(--pillar-resume))",
          foreground: "hsl(var(--pillar-foreground))",
        },
        chart: {
          1: "hsl(var(--chart-1))",
          2: "hsl(var(--chart-2))",
          3: "hsl(var(--chart-3))",
          4: "hsl(var(--chart-4))",
          5: "hsl(var(--chart-5))",
          grid: "hsl(var(--chart-grid))",
        },
      },
      fontFamily: {
        sans: ["var(--font-sans)"],
        mono: ["var(--font-mono)"],
      },
      fontSize: {
        "2xs": ["0.6875rem", { lineHeight: "1.3", fontWeight: "600", letterSpacing: "0.04em" }],
        xs: ["0.75rem", { lineHeight: "1.4" }],
        sm: ["0.875rem", { lineHeight: "1.45" }],
        base: ["1rem", { lineHeight: "1.5" }],
        lg: ["1.125rem", { lineHeight: "1.4" }],
        h3: ["1.25rem", { lineHeight: "1.3", fontWeight: "600" }],
        h2: ["1.5rem", { lineHeight: "1.25", fontWeight: "600", letterSpacing: "-0.011em" }],
        h1: ["1.875rem", { lineHeight: "1.2", fontWeight: "700", letterSpacing: "-0.011em" }],
        display: ["2.5rem", { lineHeight: "1.1", fontWeight: "700", letterSpacing: "-0.02em" }],
      },
      borderRadius: {
        sm: "calc(var(--radius) - 4px)",
        md: "calc(var(--radius) - 2px)",
        lg: "var(--radius)",
        xl: "calc(var(--radius) + 6px)",
        "2xl": "calc(var(--radius) + 14px)",
      },
      boxShadow: {
        "elevation-1": "0 1px 2px hsl(222 40% 2% / 0.4)",
        "elevation-2": "0 4px 16px hsl(222 40% 2% / 0.5)",
        "elevation-3": "0 16px 48px hsl(222 40% 1% / 0.6)",
        "glow-primary": "0 0 0 1px hsl(var(--primary) / 0.4), 0 0 24px hsl(var(--primary) / 0.25)",
      },
      zIndex: {
        sticky: "10",
        nav: "20",
        dropdown: "30",
        overlay: "40",
        modal: "50",
        toast: "60",
      },
      keyframes: {
        "fade-in": { from: { opacity: "0" }, to: { opacity: "1" } },
        "slide-up": {
          from: { opacity: "0", transform: "translateY(8px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        "scale-in": {
          from: { opacity: "0", transform: "scale(0.96)" },
          to: { opacity: "1", transform: "scale(1)" },
        },
        shimmer: { "100%": { transform: "translateX(100%)" } },
      },
      animation: {
        "fade-in": "fade-in 160ms ease-out",
        "slide-up": "slide-up 200ms cubic-bezier(0.22, 1, 0.36, 1)",
        "scale-in": "scale-in 160ms cubic-bezier(0.22, 1, 0.36, 1)",
        shimmer: "shimmer 1.6s infinite",
      },
    },
  },
  plugins: [animate],
};

export default config;
