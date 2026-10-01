"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ChevronDown, LogOut, Settings } from "lucide-react";
import Avatar from "@/components/Avatar";
import Logo from "@/components/Logo";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export const HEADER_HEIGHT_PX = 56; // keep in sync with h-14

export default function AppHeader() {
  const { user, logout, loading } = useAuth();
  const router = useRouter();

  if (loading || !user) return null;

  const displayName = user.name || user.email;

  const handleSignOut = async () => {
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

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="flex items-center gap-2 px-2 py-1 rounded-lg hover:bg-gray-100 transition-colors data-[state=open]:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
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
            <ChevronDown className="w-4 h-4 text-gray-500" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-56">
          <DropdownMenuLabel className="normal-case tracking-normal font-normal text-gray-500 truncate">
            {user.email}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => router.push("/settings")}>
            <Settings /> Settings
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={handleSignOut}>
            <LogOut /> Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
