import { apiClient } from "./api-client";
import { getApiUrl } from "@/config/api";
import type { User } from "./types";

export type { User };

export class AuthAPI {
  static async getCurrentUser(): Promise<User | null> {
    const { data, error } = await apiClient.GET("/api/auth/me");
    if (error || !data) return null;
    return data.user;
  }

  static async logout(): Promise<void> {
    const { error } = await apiClient.POST("/api/auth/logout");
    if (error) throw new Error(error.message || "Logout failed");
  }

  /** Full-page navigation target that starts Google sign-in. */
  static loginUrl(redirect?: string): string {
    const params = new URLSearchParams();
    if (redirect) params.set("redirect", redirect);
    const qs = params.toString();
    return getApiUrl(`/api/auth/google/login${qs ? `?${qs}` : ""}`);
  }
}
