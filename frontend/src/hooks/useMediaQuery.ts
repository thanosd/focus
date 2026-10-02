"use client";

import { useEffect, useState } from "react";

/**
 * Reactive `window.matchMedia` check. Returns `fallback` during server
 * rendering and the first client render, then tracks the real value.
 */
export function useMediaQuery(query: string, fallback = false): boolean {
  const [matches, setMatches] = useState(fallback);
  useEffect(() => {
    const mql = window.matchMedia(query);
    const update = () => setMatches(mql.matches);
    update();
    mql.addEventListener("change", update);
    return () => mql.removeEventListener("change", update);
  }, [query]);
  return matches;
}

/** Tailwind's `md` breakpoint and up: the docked-column layout. */
export function useIsDesktop(): boolean {
  return useMediaQuery("(min-width: 768px)", true);
}
