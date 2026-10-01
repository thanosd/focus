"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useCounts } from "@/contexts/CountsContext";
import { useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  ChevronsLeft,
  ChevronsRight,
  Flag,
  FolderKanban,
  Inbox,
  RefreshCw,
  Settings,
  Tag,
  type LucideIcon,
} from "lucide-react";
import { HEADER_HEIGHT_PX } from "@/components/AppHeader";
import { cn } from "@/lib/utils";

type Tone = "blue" | "red" | "orange";

interface NavItem {
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

function Badge({ n, tone }: { n: number; tone: Tone }) {
  return (
    <span
      className={cn(
        "ml-auto text-xs font-semibold px-2 py-0.5 rounded-full",
        TONES[tone],
      )}
    >
      {n}
    </span>
  );
}

export default function Sidebar() {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const { user, loading } = useAuth();
  const { counts } = useCounts();
  const pathname = usePathname();

  if (loading || !user) return null;

  const widthClass = isCollapsed ? "w-16" : "w-60";

  const items: NavItem[] = [
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
      href: "/review",
      label: "Review",
      icon: RefreshCw,
      badge: counts?.review_due,
      badgeTone: "blue",
    },
    { href: "/settings", label: "Settings", icon: Settings },
  ];

  const dueTotal = (counts?.overdue ?? 0) + (counts?.due_soon ?? 0);
  const Toggle = isCollapsed ? ChevronsRight : ChevronsLeft;

  return (
    <>
      <div
        className={cn(
          "fixed left-0 bg-white border-r border-gray-200 transition-all duration-300 z-30",
          widthClass,
        )}
        style={{
          top: `${HEADER_HEIGHT_PX}px`,
          height: `calc(100vh - ${HEADER_HEIGHT_PX}px)`,
        }}
      >
        <div
          className={cn(
            "flex border-b border-gray-200 p-3",
            isCollapsed ? "justify-center" : "justify-end",
          )}
        >
          <button
            type="button"
            onClick={() => setIsCollapsed(!isCollapsed)}
            className="p-2 rounded-lg hover:bg-gray-100 transition-colors text-gray-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            aria-label={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            <Toggle className="w-5 h-5" />
          </button>
        </div>

        <nav className={cn("space-y-1", isCollapsed ? "px-2 py-4" : "p-3")}>
          {items.map((item) => {
            const active =
              pathname === item.href || pathname.startsWith(item.href + "/");
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                title={item.label}
                className={cn(
                  "flex items-center gap-3 py-2 rounded-lg transition-colors",
                  active
                    ? "bg-blue-50 text-blue-600 font-medium"
                    : "text-gray-700 hover:bg-gray-100",
                  isCollapsed ? "px-0 justify-center" : "px-3",
                )}
              >
                <Icon
                  className={cn(
                    "flex-shrink-0",
                    isCollapsed ? "w-6 h-6" : "w-5 h-5",
                  )}
                  strokeWidth={1.75}
                />
                {!isCollapsed && <span>{item.label}</span>}
                {!isCollapsed && item.badge ? (
                  <Badge n={item.badge} tone={item.badgeTone ?? "blue"} />
                ) : null}
              </Link>
            );
          })}
          {!isCollapsed && dueTotal > 0 && (
            <div className="px-3 pt-3 text-xs text-gray-500 flex items-center gap-2">
              <span>Due</span>
              {counts && counts.overdue > 0 && (
                <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-red-100 text-red-700">
                  {counts.overdue} overdue
                </span>
              )}
              {counts && counts.due_soon > 0 && (
                <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-orange-100 text-orange-700">
                  {counts.due_soon} soon
                </span>
              )}
            </div>
          )}
        </nav>
      </div>

      <div
        className={cn("flex-shrink-0 transition-all duration-300", widthClass)}
      />
    </>
  );
}
