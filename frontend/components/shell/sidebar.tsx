"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { PILLAR_NAV, PRIMARY_NAV, UTILITY_NAV, type NavItem } from "@/lib/nav";
import { useUiStore } from "@/lib/store/ui";
import { cn } from "@/lib/utils";

function NavLink({
  item,
  collapsed,
  active,
}: {
  item: NavItem;
  collapsed: boolean;
  active: boolean;
}) {
  const { icon: Icon, pillar } = item;
  const accent = pillar ? `hsl(var(--pillar-${pillar}))` : "hsl(var(--primary))";

  return (
    <Link
      href={item.href}
      title={collapsed ? item.label : undefined}
      aria-current={active ? "page" : undefined}
      className={cn(
        "group/navlink relative flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium",
        "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        collapsed && "justify-center px-0",
        active
          ? "bg-muted text-foreground"
          : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
      )}
    >
      {active && (
        <span
          aria-hidden
          className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-full"
          style={{ backgroundColor: accent }}
        />
      )}
      <Icon
        className="size-5 shrink-0"
        style={pillar ? { color: accent } : undefined}
        aria-hidden
      />
      {!collapsed && <span className="truncate">{item.label}</span>}
    </Link>
  );
}

export function Sidebar() {
  const pathname = usePathname();
  const collapsed = useUiStore((s) => s.sidebarCollapsed);

  const isActive = (href: string) =>
    pathname === href || pathname.startsWith(`${href}/`);

  return (
    <aside
      aria-label="Primary"
      data-collapsed={collapsed}
      className={cn(
        "z-nav hidden shrink-0 border-r border-border bg-surface md:flex md:flex-col",
        "transition-[width] duration-200",
        collapsed ? "w-16" : "w-[260px]",
      )}
    >
      <nav className="flex flex-1 flex-col gap-1 overflow-y-auto p-3">
        {PRIMARY_NAV.map((item) => (
          <NavLink
            key={item.href}
            item={item}
            collapsed={collapsed}
            active={isActive(item.href)}
          />
        ))}

        <div className="my-2 px-3">
          {!collapsed ? (
            <span className="text-2xs uppercase text-muted-foreground">Pillars</span>
          ) : (
            <span className="block h-px bg-border" aria-hidden />
          )}
        </div>

        {PILLAR_NAV.map((item) => (
          <NavLink
            key={item.href}
            item={item}
            collapsed={collapsed}
            active={isActive(item.href)}
          />
        ))}

        <div className="mt-auto flex flex-col gap-1 pt-2">
          <span className="my-1 block h-px bg-border" aria-hidden />
          {UTILITY_NAV.map((item) => (
            <NavLink
              key={item.href}
              item={item}
              collapsed={collapsed}
              active={isActive(item.href)}
            />
          ))}
        </div>
      </nav>
    </aside>
  );
}
