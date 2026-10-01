"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Task } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useToast } from "@/contexts/ToastContext";
import { fromDatetimeLocal, toDatetimeLocal } from "@/lib/dates";
import { useEffect, useRef, useState } from "react";
import Spinner from "@/components/Spinner";

const QUICK: { label: string; input: string }[] = [
  { label: "+1 day", input: "1d" },
  { label: "+1 week", input: "1w" },
  { label: "+1 month", input: "1m" },
  { label: "Tomorrow", input: "tomorrow" },
  { label: "Next Monday", input: "next monday" },
];

interface DeferMenuProps {
  task: Task;
  onUpdated: (task: Task) => void;
  /** Render inline (inspector) instead of as a popover. */
  inline?: boolean;
  onClose?: () => void;
}

/**
 * Defer controls: quick buttons, a natural-language input (parsed by the
 * backend, AI-assisted) and an explicit datetime picker.
 */
export default function DeferMenu({
  task,
  onUpdated,
  inline = false,
  onClose,
}: DeferMenuProps) {
  const { timezone } = useAuth();
  const { toast } = useToast();
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [explicit, setExplicit] = useState(
    toDatetimeLocal(task.defer_until, timezone),
  );
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setExplicit(toDatetimeLocal(task.defer_until, timezone));
  }, [task.defer_until, timezone]);

  useEffect(() => {
    if (inline || !onClose) return;
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [inline, onClose]);

  const send = async (body: {
    input?: string;
    until?: string;
    clear?: boolean;
  }) => {
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/tasks/{taskId}/defer", {
      params: { path: { taskId: task.id } },
      body: { ...body, timezone },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't defer task"), "error");
      return;
    }
    onUpdated(data);
    setText("");
    if (body.clear) toast("Defer date cleared");
    onClose?.();
  };

  const body = (
    <div ref={ref} className="space-y-3">
      <div className="flex flex-wrap gap-1.5">
        {QUICK.map((q) => (
          <button
            key={q.input}
            type="button"
            disabled={busy}
            onClick={() => send({ input: q.input })}
            className="text-xs px-2.5 py-1 rounded-md border border-gray-200 bg-white hover:bg-gray-50 text-gray-700 disabled:opacity-50"
          >
            {q.label}
          </button>
        ))}
        <button
          type="button"
          disabled={busy || !task.defer_until}
          onClick={() => send({ clear: true })}
          className="text-xs px-2.5 py-1 rounded-md border border-gray-200 bg-white hover:bg-gray-50 text-gray-500 disabled:opacity-40"
        >
          Clear
        </button>
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (text.trim()) send({ input: text.trim() });
        }}
        className="flex items-center gap-2"
      >
        <input
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Defer until… e.g. 2w, friday, mid october"
          className="flex-1 text-sm border border-gray-300 rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
          disabled={busy}
          autoFocus={!inline}
        />
        <button
          type="submit"
          disabled={busy || !text.trim()}
          className="text-xs font-medium px-2.5 py-1.5 rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"
        >
          {busy && <Spinner className="w-3 h-3" />}
          Defer
        </button>
      </form>
      <div className="flex items-center gap-2">
        <input
          type="datetime-local"
          value={explicit}
          onChange={(e) => setExplicit(e.target.value)}
          className="flex-1 text-sm border border-gray-300 rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
          disabled={busy}
        />
        <button
          type="button"
          disabled={busy || !explicit}
          onClick={() => {
            const iso = fromDatetimeLocal(explicit, timezone);
            if (iso) send({ until: iso });
          }}
          className="text-xs font-medium px-2.5 py-1.5 rounded-md border border-gray-300 bg-white hover:bg-gray-50 text-gray-700 disabled:opacity-50"
        >
          Set
        </button>
      </div>
    </div>
  );

  if (inline) return body;

  return (
    <div className="absolute z-40 mt-1 left-0 w-80 bg-white border border-gray-200 rounded-lg shadow-lg p-3">
      <div className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
        Defer
      </div>
      {body}
    </div>
  );
}
