"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useState } from "react";
import { ChevronsLeft, ChevronsRight } from "lucide-react";
import { HEADER_HEIGHT_PX } from "@/components/AppHeader";
import { NavList } from "@/components/NavItems";
import { cn } from "@/lib/utils";

/**
 * Navigation column (desktop, md+). On phones the same entries live in the
 * drawer opened from the app bar's hamburger (see AppHeader / MobileNav).
 */
export default function Sidebar() {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const { user, loading } = useAuth();

  if (loading || !user) return null;

  const widthClass = isCollapsed ? "w-16" : "w-60";
  const Toggle = isCollapsed ? ChevronsRight : ChevronsLeft;

  return (
    <>
      <div
        className={cn(
          "hidden md:block fixed left-0 bg-white border-r border-gray-200 transition-all duration-300 z-30",
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
        <NavList collapsed={isCollapsed} />
      </div>

      {/* Spacer keeps content from sliding under the fixed column. */}
      <div
        className={cn(
          "hidden md:block flex-shrink-0 transition-all duration-300",
          widthClass,
        )}
      />
    </>
  );
}
