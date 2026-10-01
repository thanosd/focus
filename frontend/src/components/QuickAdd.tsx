"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Task } from "@/lib/types";
import { useCounts } from "@/contexts/CountsContext";
import { useToast } from "@/contexts/ToastContext";
import { useState } from "react";
import { PlusIcon } from "@/components/Icons";

interface QuickAddProps {
  projectId?: string;
  tagIds?: string[];
  flagged?: boolean;
  placeholder?: string;
  onCreated: (task: Task) => void;
}

/** Inline "type a title, press Enter" task creation bar. */
export default function QuickAdd({
  projectId,
  tagIds,
  flagged,
  placeholder = "Add a task… (Enter to save)",
  onCreated,
}: QuickAddProps) {
  const [title, setTitle] = useState("");
  const [busy, setBusy] = useState(false);
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
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
      className="flex items-center gap-2 bg-white border border-gray-200 rounded-lg px-3 py-2 shadow-sm focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500"
    >
      <PlusIcon className="w-4 h-4 text-gray-400 flex-shrink-0" />
      <input
        type="text"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={placeholder}
        className="flex-1 outline-none text-sm text-gray-900 placeholder:text-gray-400 bg-transparent"
        disabled={busy}
      />
      {title.trim() && (
        <button
          type="submit"
          disabled={busy}
          className="text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md px-2.5 py-1 disabled:opacity-50"
        >
          Add
        </button>
      )}
    </form>
  );
}
