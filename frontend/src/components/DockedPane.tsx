"use client";

import { HEADER_HEIGHT_PX } from "@/components/AppHeader";
import { Dialog, SheetContent } from "@/components/ui/dialog";
import { useIsDesktop } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

interface DockedPaneProps {
  /** Whether something is selected. */
  open: boolean;
  /** Called when the phone sheet is dismissed (swipe/Escape/overlay). */
  onClose?: () => void;
  /** Accessible name for the pane. */
  label: string;
  /** Shown in the desktop column when nothing is selected. */
  placeholder?: string;
  /**
   * Keep the desktop column populated even when `open` is false (used by
   * the project page, where the column always has the project to show and
   * `open` only drives the phone sheet).
   */
  desktopAlwaysOpen?: boolean;
  children: React.ReactNode;
}

/**
 * Properties column. On md+ it is a real layout column that sticks under
 * the header with its own scroll; the items column shrinks to fit — never a
 * modal. Below md (phones) the same content opens as a full-height sheet
 * with its own close control.
 */
export default function DockedPane({
  open,
  onClose,
  label,
  placeholder = "Select an item to see its details.",
  desktopAlwaysOpen = false,
  children,
}: DockedPaneProps) {
  // The sheet is portaled to <body>, so a `md:hidden` wrapper can't hide
  // it; decide with a real media query instead and never mount it on
  // desktop (a mounted Dialog would also lock page scrolling there).
  const isDesktop = useIsDesktop();
  return (
    <>
      <aside
        aria-label={label}
        className={cn(
          "hidden md:flex flex-col flex-shrink-0 w-[25rem] xl:w-[26rem] sticky self-start",
          "bg-white border-l border-gray-200",
        )}
        style={{
          top: `${HEADER_HEIGHT_PX}px`,
          height: `calc(100vh - ${HEADER_HEIGHT_PX}px)`,
        }}
      >
        {open || desktopAlwaysOpen ? (
          children
        ) : (
          <div className="flex-1 flex items-center justify-center p-8 text-center text-sm text-gray-400">
            {placeholder}
          </div>
        )}
      </aside>

      {!isDesktop && (
        <Dialog open={open} onOpenChange={(o) => !o && onClose?.()}>
          <SheetContent aria-label={label}>{open && children}</SheetContent>
        </Dialog>
      )}
    </>
  );
}
