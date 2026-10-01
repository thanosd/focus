"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Task } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useToast } from "@/contexts/ToastContext";
import { fromDatetimeLocal, toDatetimeLocal } from "@/lib/dates";
import { useEffect, useState } from "react";
import Spinner from "@/components/Spinner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

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
  /** Called after a successful change (popover hosts close themselves). */
  onDone?: () => void;
  autoFocus?: boolean;
}

/**
 * Defer controls: quick buttons, a natural-language input (parsed by the
 * backend, AI-assisted) and an explicit datetime picker. Layout-agnostic:
 * the inspector renders it inline, task rows put it in a Popover.
 */
export default function DeferMenu({
  task,
  onUpdated,
  onDone,
  autoFocus = false,
}: DeferMenuProps) {
  const { timezone } = useAuth();
  const { toast } = useToast();
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [explicit, setExplicit] = useState(
    toDatetimeLocal(task.defer_until, timezone),
  );

  useEffect(() => {
    setExplicit(toDatetimeLocal(task.defer_until, timezone));
  }, [task.defer_until, timezone]);

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
    onDone?.();
  };

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-1.5">
        {QUICK.map((q) => (
          <Button
            key={q.input}
            size="xs"
            disabled={busy}
            onClick={() => send({ input: q.input })}
          >
            {q.label}
          </Button>
        ))}
        <Button
          size="xs"
          variant="ghost"
          disabled={busy || !task.defer_until}
          onClick={() => send({ clear: true })}
        >
          Clear
        </Button>
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (text.trim()) send({ input: text.trim() });
        }}
        className="flex items-center gap-2"
      >
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Defer until… e.g. 2w, friday, mid october"
          disabled={busy}
          autoFocus={autoFocus}
        />
        <Button type="submit" variant="primary" disabled={busy || !text.trim()}>
          {busy && <Spinner className="w-3 h-3" />}
          Defer
        </Button>
      </form>
      <div className="flex items-center gap-2">
        <Input
          type="datetime-local"
          value={explicit}
          onChange={(e) => setExplicit(e.target.value)}
          disabled={busy}
        />
        <Button
          disabled={busy || !explicit}
          onClick={() => {
            const iso = fromDatetimeLocal(explicit, timezone);
            if (iso) send({ until: iso });
          }}
        >
          Set
        </Button>
      </div>
    </div>
  );
}
