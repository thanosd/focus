"use client";

import { useCounts } from "@/contexts/CountsContext";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  CalendarClock,
  Flag,
  FolderKanban,
  Inbox,
  RefreshCw,
  Settings,
  Tag,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

type Tone = "blue" | "red" | "orange";

export interface NavItem {
  href: string;
  label: string;
  icon: LucideIcon;
  badge?: number;
  badgeTone?: Tone;
}

const TONES: Record<Tone, string> = {
  blue: "bg-blue-100 text-blue-700",
  red: "bg-red-100 text-red-700",
  orange: "bg-orange-100 text-orange-700",
};

/** The navigation column's entries, with live badge counts. */
export function useNavItems(): NavItem[] {
  const { counts } = useCounts();
  return [
    {
      href: "/inbox",
      label: "Inbox",
      icon: Inbox,
      badge: counts?.inbox,
      badgeTone: "blue",
    },
    { href: "/projects", label: "Projects", icon: FolderKanban },
    { href: "/tags", label: "Tags", icon: Tag },
    {
      href: "/flagged",
      label: "Flagged",
      icon: Flag,
      badge: counts?.flagged,
      badgeTone: "orange",
    },
    {
      href: "/due",
      label: "Due",
      icon: CalendarClock,
      badge: (counts?.overdue ?? 0) + (counts?.due_soon ?? 0) || undefined,
      badgeTone: counts && counts.overdue > 0 ? "red" : "orange",
    },
    {
      href: "/review",
      label: "Review",
      icon: RefreshCw,
      badge: counts?.review_due,
      badgeTone: "blue",
    },
    { href: "/settings", label: "Settings", icon: Settings },
  ];
}

/**
 * Navigation list shared by the desktop navigation column and the phone
 * drawer. Rows are ≥ 44px tall when `touch` is set.
 */
export function NavList({
  collapsed = false,
  touch = false,
  onNavigate,
}: {
  collapsed?: boolean;
  touch?: boolean;
  onNavigate?: () => void;
}) {
  const items = useNavItems();
  const { counts } = useCounts();
  const pathname = usePathname();
  return (
    <nav className={cn("space-y-1", collapsed ? "px-2 py-4" : "p-3")}>
      {items.map((item) => {
        const active =
          pathname === item.href || pathname.startsWith(item.href + "/");
        const Icon = item.icon;
        return (
          <Link
            key={item.href}
            href={item.href}
            title={item.label}
            onClick={onNavigate}
            className={cn(
              "flex items-center gap-3 rounded-lg transition-colors",
              touch ? "py-3 text-base" : "py-2",
              active
                ? "bg-blue-50 text-blue-600 font-medium"
                : "text-gray-700 hover:bg-gray-100",
              collapsed ? "px-0 justify-center" : "px-3",
            )}
          >
            <Icon
              className={cn("flex-shrink-0", collapsed ? "w-6 h-6" : "w-5 h-5")}
              strokeWidth={1.75}
            />
            {!collapsed && <span>{item.label}</span>}
            {!collapsed && item.badge ? (
              <span
                className={cn(
                  "ml-auto text-xs font-semibold px-2 py-0.5 rounded-full",
                  TONES[item.badgeTone ?? "blue"],
                )}
              >
                {item.badge}
              </span>
            ) : null}
          </Link>
        );
      })}
      {!collapsed && counts && counts.overdue > 0 && (
        <Link
          href="/due"
          onClick={onNavigate}
          className="block px-3 pt-1 text-xs text-red-600 hover:underline"
        >
          {counts.overdue} overdue
        </Link>
      )}
    </nav>
  );
}
