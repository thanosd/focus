/**
 * API Configuration
 *
 * The API URL is read from the NEXT_PUBLIC_API_URL environment variable.
 * For local development, set it in .env.local; in production it is a
 * build argument on the Docker image.
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL;

if (!API_URL) {
  console.warn(
    "NEXT_PUBLIC_API_URL is not defined. API calls may fail or use relative paths. " +
      "Please set this variable in .env.local for development.",
  );
}

export const config = {
  apiUrl: API_URL || "",
} as const;

/** Build an absolute API URL for a path (used for browser navigations). */
export function getApiUrl(path: string): string {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${config.apiUrl}${normalizedPath}`;
}
