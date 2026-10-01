"use client";

import { useAuth } from "@/contexts/AuthContext";
import { AuthAPI } from "@/lib/auth";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";
import Logo from "@/components/Logo";

const ERROR_MESSAGES: Record<string, string> = {
  not_allowed:
    "This Google account isn't on the allowlist. Ask the administrator to add your email.",
  oauth_failed: "Google sign-in didn't complete. Please try again.",
  state_mismatch:
    "The sign-in session expired or was tampered with. Please try again.",
  session: "We couldn't create a session. Please try again.",
};

function LoginContent() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const errorCode = searchParams.get("error");
  const next = searchParams.get("next") || "/inbox";

  useEffect(() => {
    if (!loading && user) {
      router.replace(next.startsWith("/") ? next : "/inbox");
    }
  }, [user, loading, router, next]);

  const message = errorCode
    ? ERROR_MESSAGES[errorCode] || "Sign-in failed. Please try again."
    : null;

  return (
    <div className="min-h-[70vh] flex items-center justify-center p-8">
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-8 max-w-sm w-full text-center">
        <div className="flex justify-center mb-4">
          <Logo className="w-12 h-12" />
        </div>
        <h1 className="text-2xl font-bold text-gray-900 mb-1">Focus</h1>
        <p className="text-sm text-gray-500 mb-6">
          Projects, tasks, tags and reviews.
        </p>
        {message && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3 mb-4 text-left">
            {message}
          </div>
        )}
        <a
          href={AuthAPI.loginUrl(next)}
          className="inline-flex items-center justify-center gap-3 w-full px-4 py-2.5 rounded-lg border border-gray-300 bg-white text-gray-800 font-medium hover:bg-gray-50 transition-colors"
        >
          <svg className="w-5 h-5" viewBox="0 0 48 48" aria-hidden="true">
            <path
              fill="#EA4335"
              d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
            />
            <path
              fill="#4285F4"
              d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
            />
            <path
              fill="#FBBC05"
              d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
            />
            <path
              fill="#34A853"
              d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
            />
          </svg>
          Sign in with Google
        </a>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginContent />
    </Suspense>
  );
}
