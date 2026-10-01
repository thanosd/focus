"use client";

import { HEADER_HEIGHT_PX } from "@/components/AppHeader";
import { cn } from "@/lib/utils";

interface DockedPaneProps {
  /** Whether something is selected. */
  open: boolean;
  /** Kept for callers; closing is handled by the content's own close button. */
  onClose?: () => void;
  /** Accessible name for the pane. */
  label: string;
  /** Shown in the desktop column when nothing is selected. */
  placeholder?: string;
  /**
   * Keep the desktop column populated even when `open` is false (used by
   * the project page, where the pane always has the project to show and
   * `open` only drives the mobile sheet).
   */
  desktopAlwaysOpen?: boolean;
  children: React.ReactNode;
}

/**
 * Right-hand inspector column: a real layout column that sticks under the
 * header with its own scroll; the main column shrinks to fit. It is never a
 * modal or full-screen sheet — selecting something just fills the column.
 */
export default function DockedPane({
  open,
  label,
  placeholder = "Select an item to see its details.",
  desktopAlwaysOpen = false,
  children,
}: DockedPaneProps) {
  return (
    <>
      <aside
        aria-label={label}
        className={cn(
          "flex flex-col flex-shrink-0 w-[20rem] md:w-[25rem] xl:w-[26rem] sticky self-start",
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
    </>
  );
}
