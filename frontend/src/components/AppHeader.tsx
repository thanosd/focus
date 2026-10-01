"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import Avatar from "@/components/Avatar";
import Logo from "@/components/Logo";

export const HEADER_HEIGHT_PX = 56; // keep in sync with h-14

export default function AppHeader() {
  const { user, logout, loading } = useAuth();
  const router = useRouter();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const onClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, [menuOpen]);

  if (loading || !user) return null;

  const displayName = user.name || user.email;

  const handleSignOut = async () => {
    setMenuOpen(false);
    await logout();
    router.push("/login");
  };

  return (
    <header className="sticky top-0 z-50 flex items-center justify-between h-14 px-6 bg-white border-b border-gray-200">
      <Link href="/inbox" className="flex items-center gap-2 min-w-0">
        <Logo className="w-6 h-6 flex-shrink-0" />
        <span className="text-lg font-semibold text-gray-900 truncate">
          Focus
        </span>
      </Link>

      <div ref={menuRef} className="relative">
        <button
          onClick={() => setMenuOpen((v) => !v)}
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          className="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-gray-100 transition-colors"
        >
          <Avatar
            email={user.email}
            name={user.name}
            pictureUrl={user.picture_url}
            size="md"
          />
          <span
            className="text-sm font-medium text-gray-900 max-w-[12rem] truncate"
            title={user.email}
          >
            {displayName}
          </span>
          <svg
            className={`w-4 h-4 text-gray-500 transition-transform ${
              menuOpen ? "rotate-180" : ""
            }`}
            fill="none"
            viewBox="0 0 20 20"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M5 7l5 5 5-5"
            />
          </svg>
        </button>

        {menuOpen && (
          <div
            role="menu"
            className="absolute right-0 mt-2 w-48 bg-white border border-gray-200 rounded-lg shadow-lg py-1 z-50"
          >
            <div className="px-4 py-2 text-xs text-gray-500 truncate border-b border-gray-100">
              {user.email}
            </div>
            <Link
              href="/settings"
              role="menuitem"
              onClick={() => setMenuOpen(false)}
              className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
            >
              Settings
            </Link>
            <button
              onClick={handleSignOut}
              role="menuitem"
              className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
            >
              Sign out
            </button>
          </div>
        )}
      </div>
    </header>
  );
}
