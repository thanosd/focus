"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Task } from "@/lib/types";
import { useCounts } from "@/contexts/CountsContext";
import { useToast } from "@/contexts/ToastContext";
import { useRef, useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

interface QuickAddProps {
  projectId?: string;
  tagIds?: string[];
  flagged?: boolean;
  placeholder?: string;
  onCreated: (task: Task) => void;
}

/**
 * Inline "type a title, press Enter" task creation bar. Enter saves, clears
 * the field and keeps focus so the next capture can start immediately; it
 * never opens the created task.
 */
export default function QuickAdd({
  projectId,
  tagIds,
  flagged,
  placeholder = "Add a task… (Enter to save)",
  onCreated,
}: QuickAddProps) {
  const [title, setTitle] = useState("");
  const [busy, setBusy] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const { refreshCounts } = useCounts();
  const { toast } = useToast();

  const submit = async () => {
    const t = title.trim();
    if (!t || busy) return;
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/tasks", {
      body: {
        title: t,
        project_id: projectId,
        tag_ids: tagIds,
        flagged: flagged || undefined,
      },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Failed to create task"), "error");
      return;
    }
    setTitle("");
    onCreated(data);
    refreshCounts();
    inputRef.current?.focus();
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
      className="flex items-center gap-2 bg-white border border-gray-300 rounded-md px-3 h-10 shadow-sm focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500"
    >
      <Plus className="w-4 h-4 text-gray-400 flex-shrink-0" />
      <input
        ref={inputRef}
        type="text"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={placeholder}
        className="flex-1 outline-none text-sm text-gray-900 placeholder:text-gray-400 bg-transparent"
        readOnly={busy}
      />
      {title.trim() && (
        <Button type="submit" variant="primary" size="xs" disabled={busy}>
          Add
        </Button>
      )}
    </form>
  );
}
