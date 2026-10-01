"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useCounts } from "@/contexts/CountsContext";
import { useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { HEADER_HEIGHT_PX } from "@/components/AppHeader";

interface NavItem {
  href: string;
  label: string;
  icon: React.ReactNode;
  badge?: number;
  badgeTone?: "blue" | "red" | "orange";
}

function Badge({ n, tone }: { n: number; tone: "blue" | "red" | "orange" }) {
  const cls = {
    blue: "bg-blue-100 text-blue-700",
    red: "bg-red-100 text-red-700",
    orange: "bg-orange-100 text-orange-700",
  }[tone];
  return (
    <span
      className={`ml-auto text-xs font-semibold px-2 py-0.5 rounded-full ${cls}`}
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

  const iconSize = isCollapsed ? "w-7 h-7" : "w-5 h-5";
  const widthClass = isCollapsed ? "w-16" : "w-60";
  const stroke = {
    fill: "none",
    viewBox: "0 0 24 24",
    stroke: "currentColor",
    strokeWidth: 2,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
  };

  const items: NavItem[] = [
    {
      href: "/inbox",
      label: "Inbox",
      badge: counts?.inbox,
      badgeTone: "blue",
      icon: (
        <svg className={`${iconSize} flex-shrink-0`} {...stroke}>
          <path d="M3 12l2-7h14l2 7v7a1 1 0 01-1 1H4a1 1 0 01-1-1v-7z" />
          <path d="M3 12h5l2 3h4l2-3h5" />
        </svg>
      ),
    },
    {
      href: "/projects",
      label: "Projects",
      icon: (
        <svg className={`${iconSize} flex-shrink-0`} {...stroke}>
          <path d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
        </svg>
      ),
    },
    {
      href: "/tags",
      label: "Tags",
      icon: (
        <svg className={`${iconSize} flex-shrink-0`} {...stroke}>
          <path d="M20 12l-8 8-9-9V4h7l10 10z" />
          <circle cx="7.5" cy="7.5" r="1" />
        </svg>
      ),
    },
    {
      href: "/flagged",
      label: "Flagged",
      badge: counts?.flagged,
      badgeTone: "orange",
      icon: (
        <svg className={`${iconSize} flex-shrink-0`} {...stroke}>
          <path d="M5 21V4a1 1 0 011-1h9l1 2h4v10h-6l-1-2H5" />
        </svg>
      ),
    },
    {
      href: "/review",
      label: "Review",
      badge: counts?.review_due,
      badgeTone: "blue",
      icon: (
        <svg className={`${iconSize} flex-shrink-0`} {...stroke}>
          <path d="M4 4v6h6" />
          <path d="M20 20v-6h-6" />
          <path d="M20 9a8 8 0 00-14.5-3.5L4 10" />
          <path d="M4 15a8 8 0 0014.5 3.5L20 14" />
        </svg>
      ),
    },
    {
      href: "/settings",
      label: "Settings",
      icon: (
        <svg className={`${iconSize} flex-shrink-0`} {...stroke}>
          <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      ),
    },
  ];

  const dueTotal = (counts?.overdue ?? 0) + (counts?.due_soon ?? 0);

  return (
    <>
      <div
        className={`fixed left-0 bg-white border-r border-gray-200 transition-all duration-300 z-30 ${widthClass}`}
        style={{
          top: `${HEADER_HEIGHT_PX}px`,
          height: `calc(100vh - ${HEADER_HEIGHT_PX}px)`,
        }}
      >
        <div
          className={`flex border-b border-gray-200 ${
            isCollapsed ? "justify-center p-3" : "justify-end p-3"
          }`}
        >
          <button
            onClick={() => setIsCollapsed(!isCollapsed)}
            className="p-2 rounded-lg hover:bg-gray-100 transition-colors"
            aria-label={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            <svg className="w-5 h-5 text-gray-600" {...stroke}>
              {isCollapsed ? (
                <path d="M13 5l7 7-7 7M5 5l7 7-7 7" />
              ) : (
                <path d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
              )}
            </svg>
          </button>
        </div>

        <nav className={`space-y-1 ${isCollapsed ? "px-2 py-4" : "p-3"}`}>
          {items.map((item) => {
            const active =
              pathname === item.href || pathname.startsWith(item.href + "/");
            return (
              <Link
                key={item.href}
                href={item.href}
                title={item.label}
                className={`flex items-center gap-3 py-2 rounded-lg transition-colors ${
                  active
                    ? "bg-blue-50 text-blue-600 font-medium"
                    : "text-gray-700 hover:bg-gray-100"
                } ${isCollapsed ? "px-0 justify-center" : "px-3"}`}
              >
                {item.icon}
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
        className={`flex-shrink-0 transition-all duration-300 ${widthClass}`}
      />
    </>
  );
}
