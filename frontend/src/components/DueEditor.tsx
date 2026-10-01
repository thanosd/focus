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

interface DueEditorProps {
  task: Task;
  onUpdated: (task: Task) => void;
}

/** Due date editor: natural-language input plus a datetime picker. */
export default function DueEditor({ task, onUpdated }: DueEditorProps) {
  const { timezone } = useAuth();
  const { toast } = useToast();
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [explicit, setExplicit] = useState(
    toDatetimeLocal(task.due_at, timezone),
  );
  const [interpretation, setInterpretation] = useState<string | null>(null);

  useEffect(() => {
    setExplicit(toDatetimeLocal(task.due_at, timezone));
  }, [task.due_at, timezone]);

  const patch = async (due_at: string | null) => {
    setBusy(true);
    const { data, error } = await apiClient.PATCH("/api/tasks/{taskId}", {
      params: { path: { taskId: task.id } },
      body: { due_at },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't update due date"), "error");
      return;
    }
    onUpdated(data);
  };

  const parseAndSet = async () => {
    const input = text.trim();
    if (!input) return;
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/dates/parse", {
      body: { input, kind: "due", timezone },
    });
    if (error || !data) {
      setBusy(false);
      toast(errorMessage(error, "Couldn't understand that date"), "error");
      return;
    }
    setInterpretation(data.interpretation);
    setText("");
    await patch(data.resolved_at);
  };

  return (
    <div className="space-y-2">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          parseAndSet();
        }}
        className="flex items-center gap-2"
      >
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Due… e.g. friday, end of month, in 3 days"
          disabled={busy}
        />
        <Button type="submit" variant="primary" disabled={busy || !text.trim()}>
          {busy && <Spinner className="w-3 h-3" />}
          Set
        </Button>
      </form>
      {interpretation && (
        <p className="text-xs text-gray-500">Interpreted as {interpretation}</p>
      )}
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
            if (iso) patch(iso);
          }}
        >
          Set
        </Button>
        <Button
          variant="ghost"
          disabled={busy || !task.due_at}
          onClick={() => patch(null)}
        >
          Clear
        </Button>
      </div>
    </div>
  );
}
