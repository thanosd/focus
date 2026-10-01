"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function Home() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    router.replace(user ? "/inbox" : "/login");
  }, [user, loading, router]);

  return (
    <main className="min-h-screen p-8 flex items-center justify-center">
      <div className="text-gray-600">Loading...</div>
    </main>
  );
}
