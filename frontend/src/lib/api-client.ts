/**
 * Typed API Client
 *
 * Type-safe client built on openapi-fetch and the types generated from
 * api/openapi.yaml. Every request includes the session cookie.
 */

import createClient from "openapi-fetch";
import type { paths } from "./api-types";
import { config } from "@/config/api";

export const apiClient = createClient<paths>({
  baseUrl: config.apiUrl,
  credentials: "include",
  headers: {
    "Content-Type": "application/json",
  },
});

let csrfToken: string | null = null;
let fetchingCsrfToken: Promise<string> | null = null;

async function getCsrfToken(): Promise<string> {
  if (csrfToken) return csrfToken;
  if (fetchingCsrfToken) return fetchingCsrfToken;

  fetchingCsrfToken = fetch(`${config.apiUrl}/api/csrf`, {
    credentials: "include",
    cache: "no-store",
  })
    .then(async (res) => {
      if (!res.ok) {
        throw new Error(`Failed to fetch CSRF token: ${res.status}`);
      }
      return res.json();
    })
    .then((data) => {
      csrfToken = data.csrfToken;
      return csrfToken as string;
    })
    .catch((err) => {
      fetchingCsrfToken = null;
      throw err;
    });

  return fetchingCsrfToken;
}

// The backend's CSRF middleware (filippo.io/csrf) relies on Fetch metadata
// headers rather than tokens; /api/csrf returns an empty token. We still
// prime it and forward it when present for compatibility.
apiClient.use({
  onRequest: async ({ request }) => {
    if (request.url.includes("/api/csrf")) return request;
    if (!csrfToken) {
      await getCsrfToken().catch(() => {
        // Continue without a token.
      });
    }
    if (csrfToken) {
      request.headers.set("X-CSRF-Token", csrfToken);
    }
    return request;
  },
});

/** Extract a human-readable message from an openapi-fetch error value. */
export function errorMessage(error: unknown, fallback = "Request failed"): string {
  if (!error) return fallback;
  if (typeof error === "string") return error;
  if (typeof error === "object" && "message" in error) {
    const m = (error as { message?: unknown }).message;
    if (typeof m === "string" && m) return m;
  }
  return fallback;
}

export type { components, paths } from "./api-types";
