"use client";

import { AuthAPI, type User } from "@/lib/auth";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

interface AuthContextType {
  user: User | null;
  loading: boolean;
  logout: () => Promise<void>;
  refetchUser: () => Promise<void>;
  /** Timezone to render dates in: the user's preference, else the browser's. */
  timezone: string;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

function browserTz(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const refetchUser = useCallback(async () => {
    try {
      const u = await AuthAPI.getCurrentUser();
      setUser(u);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refetchUser();
  }, [refetchUser]);

  const logout = useCallback(async () => {
    try {
      await AuthAPI.logout();
    } finally {
      setUser(null);
    }
  }, []);

  const timezone = user?.timezone || browserTz();

  return (
    <AuthContext.Provider
      value={{ user, loading, logout, refetchUser, timezone }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
