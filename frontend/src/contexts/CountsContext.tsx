"use client";

import { apiClient } from "@/lib/api-client";
import type { Counts } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

interface CountsContextType {
  counts: Counts | null;
  /** Re-fetch sidebar badge counts. Call after any mutation. */
  refreshCounts: () => Promise<void>;
}

const CountsContext = createContext<CountsContextType | undefined>(undefined);

export function CountsProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const [counts, setCounts] = useState<Counts | null>(null);

  const refreshCounts = useCallback(async () => {
    if (!user) {
      setCounts(null);
      return;
    }
    const { data } = await apiClient.GET("/api/counts");
    if (data) setCounts(data);
  }, [user]);

  useEffect(() => {
    refreshCounts();
  }, [refreshCounts]);

  return (
    <CountsContext.Provider value={{ counts, refreshCounts }}>
      {children}
    </CountsContext.Provider>
  );
}

export function useCounts() {
  const context = useContext(CountsContext);
  if (context === undefined) {
    throw new Error("useCounts must be used within a CountsProvider");
  }
  return context;
}
