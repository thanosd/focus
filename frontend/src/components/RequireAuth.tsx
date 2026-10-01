"use client";

import { useAuth } from "@/contexts/AuthContext";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Gate for app pages: redirects signed-out visitors to /login (remembering
 * where they were) and renders children only once a user is loaded.
 */
export default function RequireAuth({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !user) {
      const next = pathname && pathname !== "/" ? pathname : "/inbox";
      router.replace(`/login?next=${encodeURIComponent(next)}`);
    }
  }, [user, loading, router, pathname]);

  if (loading || !user) {
    return (
      <div className="p-8 flex items-center justify-center text-gray-600">
        Loading...
      </div>
    );
  }

  return <>{children}</>;
}
