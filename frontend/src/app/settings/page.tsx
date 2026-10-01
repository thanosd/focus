"use client";

import RequireAuth from "@/components/RequireAuth";
import { apiClient, errorMessage } from "@/lib/api-client";
import { config } from "@/config/api";
import type { ApiToken } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useToast } from "@/contexts/ToastContext";
import { dayLabel } from "@/lib/dates";
import { useEffect, useMemo, useState } from "react";

const FALLBACK_TIMEZONES = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Toronto",
  "America/Sao_Paulo",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Europe/Athens",
  "Asia/Tokyo",
  "Asia/Singapore",
  "Asia/Kolkata",
  "Australia/Sydney",
];

function timezoneOptions(current: string): string[] {
  let list: string[] = FALLBACK_TIMEZONES;
  try {
    const intl = Intl as unknown as {
      supportedValuesOf?: (key: string) => string[];
    };
    if (typeof intl.supportedValuesOf === "function") {
      list = intl.supportedValuesOf("timeZone");
    }
  } catch {
    // keep fallback
  }
  return list.includes(current) ? list : [current, ...list];
}

function SettingsContent() {
  const { user, timezone, refetchUser } = useAuth();
  const { toast } = useToast();
  const [tz, setTz] = useState(timezone);
  const [savingTz, setSavingTz] = useState(false);
  const [tokens, setTokens] = useState<ApiToken[]>([]);
  const [tokenName, setTokenName] = useState("");
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const options = useMemo(() => timezoneOptions(tz), [tz]);

  useEffect(() => {
    setTz(timezone);
  }, [timezone]);

  useEffect(() => {
    (async () => {
      const { data } = await apiClient.GET("/api/api-tokens");
      if (data) setTokens(data);
    })();
  }, []);

  const saveTz = async (value: string) => {
    setTz(value);
    setSavingTz(true);
    const { error } = await apiClient.PATCH("/api/user-preferences", {
      body: { timezone: value },
    });
    setSavingTz(false);
    if (error) {
      toast(errorMessage(error, "Couldn't save timezone"), "error");
      return;
    }
    await refetchUser();
    toast("Timezone saved", "success");
  };

  const createToken = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!tokenName.trim()) return;
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/api-tokens", {
      body: { name: tokenName.trim() },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't create token"), "error");
      return;
    }
    setTokens((t) => [data.api_token, ...t]);
    setNewSecret(data.token);
    setTokenName("");
  };

  const revoke = async (token: ApiToken) => {
    if (!window.confirm(`Revoke "${token.name}"? Clients using it will stop working.`))
      return;
    const { error } = await apiClient.DELETE("/api/api-tokens/{tokenId}", {
      params: { path: { tokenId: token.id } },
    });
    if (error) {
      toast(errorMessage(error, "Couldn't revoke token"), "error");
      return;
    }
    setTokens((t) => t.filter((x) => x.id !== token.id));
  };

  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      toast("Copied", "success");
    } catch {
      toast("Copy failed — select the text manually", "error");
    }
  };

  const mcpUrl = `${config.apiUrl}/mcp`;
  const snippet = `claude mcp add --transport http focus ${mcpUrl} --header "Authorization: Bearer ${newSecret ?? "<token>"}"`;

  return (
    <div className="p-6 md:p-8 max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
        <p className="text-sm text-gray-500 mt-0.5">
          Signed in as {user?.email}
        </p>
      </div>

      <section className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-1">Timezone</h2>
        <p className="text-sm text-gray-500 mb-4">
          Used to display dates and to resolve phrases like &ldquo;tomorrow&rdquo;
          or &ldquo;next Monday&rdquo; when deferring tasks.
        </p>
        <div className="flex items-center gap-3">
          <select
            value={tz}
            onChange={(e) => saveTz(e.target.value)}
            disabled={savingTz}
            className="border border-gray-300 rounded-md px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {options.map((z) => (
              <option key={z} value={z}>
                {z}
              </option>
            ))}
          </select>
          {savingTz && <span className="text-xs text-gray-500">Saving…</span>}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-1">
          API tokens &amp; MCP
        </h2>
        <p className="text-sm text-gray-500 mb-4">
          Personal tokens let Claude (or any MCP client) read and update your
          tasks through the Focus MCP server. Each token is shown once.
        </p>

        <form onSubmit={createToken} className="flex items-center gap-2 mb-4">
          <input
            type="text"
            value={tokenName}
            onChange={(e) => setTokenName(e.target.value)}
            placeholder="Token name, e.g. Claude Code on laptop"
            className="flex-1 text-sm border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            disabled={busy || !tokenName.trim()}
            className="text-sm font-medium px-3 py-2 rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
          >
            Create token
          </button>
        </form>

        {newSecret && (
          <div className="mb-4 bg-amber-50 border border-amber-200 rounded-lg p-4 space-y-2">
            <div className="text-sm font-medium text-amber-900">
              Copy this token now. It will not be shown again.
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs bg-white border border-amber-200 rounded px-2 py-1.5 break-all select-all">
                {newSecret}
              </code>
              <button
                type="button"
                onClick={() => copy(newSecret)}
                className="text-xs font-medium px-2.5 py-1.5 rounded-md border border-amber-300 bg-white hover:bg-amber-100"
              >
                Copy
              </button>
            </div>
            <button
              type="button"
              onClick={() => setNewSecret(null)}
              className="text-xs text-amber-800 hover:underline"
            >
              I&apos;ve saved it
            </button>
          </div>
        )}

        {tokens.length === 0 ? (
          <div className="text-sm text-gray-500 border border-dashed border-gray-200 rounded-lg p-4 text-center">
            No tokens yet.
          </div>
        ) : (
          <div className="border border-gray-200 rounded-lg divide-y divide-gray-100">
            {tokens.map((t) => (
              <div key={t.id} className="flex items-center gap-3 px-4 py-2.5 text-sm">
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-gray-900 truncate">{t.name}</div>
                  <div className="text-xs text-gray-500">
                    <code>{t.token_prefix}…</code> · created{" "}
                    {dayLabel(t.created_at, timezone)}
                    {t.last_used_at
                      ? ` · last used ${dayLabel(t.last_used_at, timezone)}`
                      : " · never used"}
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => revoke(t)}
                  className="text-xs text-red-500 hover:text-red-700 px-2 py-1 rounded-md hover:bg-red-50"
                >
                  Revoke
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="mt-5 space-y-2">
          <div className="text-sm font-medium text-gray-900">
            Connect Claude Code
          </div>
          <div className="flex items-start gap-2">
            <pre className="flex-1 text-xs bg-gray-900 text-gray-100 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap break-all">
              {snippet}
            </pre>
            <button
              type="button"
              onClick={() => copy(snippet)}
              className="text-xs font-medium px-2.5 py-1.5 rounded-md border border-gray-300 bg-white hover:bg-gray-50"
            >
              Copy
            </button>
          </div>
          <p className="text-xs text-gray-500">
            Any MCP client that supports the Streamable HTTP transport can use
            the same URL (<code>{mcpUrl}</code>) with an{" "}
            <code>Authorization: Bearer &lt;token&gt;</code> header.
          </p>
        </div>
      </section>
    </div>
  );
}

export default function SettingsPage() {
  return (
    <RequireAuth>
      <SettingsContent />
    </RequireAuth>
  );
}
