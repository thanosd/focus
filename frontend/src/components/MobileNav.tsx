"use client";

import { useState } from "react";
import { Menu } from "lucide-react";
import Logo from "@/components/Logo";
import { NavList } from "@/components/NavItems";
import {
  Dialog,
  DialogCloseButton,
  DialogTitle,
  DrawerContent,
} from "@/components/ui/dialog";

/** Hamburger + left drawer holding the navigation column on phones. */
export default function MobileNav() {
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label="Open navigation"
        className="md:hidden -ml-2 p-2.5 rounded-lg text-gray-700 hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
      >
        <Menu className="w-6 h-6" />
      </button>
      <DrawerContent aria-label="Navigation">
        <div className="flex items-center justify-between h-14 px-4 border-b border-gray-200">
          <DialogTitle className="flex items-center gap-2">
            <Logo className="w-6 h-6" />
            Focus
          </DialogTitle>
          <DialogCloseButton />
        </div>
        <NavList touch onNavigate={() => setOpen(false)} />
      </DrawerContent>
    </Dialog>
  );
}
