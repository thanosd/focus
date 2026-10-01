"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Project } from "@/lib/types";
import { useToast } from "@/contexts/ToastContext";
import { useState } from "react";

interface ProjectFormProps {
  projects: Project[];
  defaultParentId?: string;
  onCreated: (project: Project) => void;
  onCancel: () => void;
}

/** "New project" form. */
export default function ProjectForm({
  projects,
  defaultParentId,
  onCreated,
  onCancel,
}: ProjectFormProps) {
  const { toast } = useToast();
  const [name, setName] = useState("");
  const [parentId, setParentId] = useState(defaultParentId ?? "");
  const [sequential, setSequential] = useState(false);
  const [interval, setInterval] = useState(7);
  const [busy, setBusy] = useState(false);
  const topLevel = projects
    .filter((p) => !p.parent_id && p.status !== "dropped" && p.status !== "completed")
    .sort((a, b) => a.name.localeCompare(b.name));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/projects", {
      body: {
        name: name.trim(),
        parent_id: parentId || undefined,
        sequential,
        review_interval_days: interval,
      },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't create project"), "error");
      return;
    }
    onCreated(data);
  };

  return (
    <form
      onSubmit={submit}
      className="bg-white border border-gray-200 rounded-lg p-4 space-y-3 shadow-sm"
    >
      <div className="text-sm font-semibold text-gray-900">New project</div>
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Project name"
        autoFocus
        className="w-full text-sm border border-gray-300 rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
      <select
        value={parentId}
        onChange={(e) => setParentId(e.target.value)}
        className="w-full text-sm border border-gray-300 rounded-md px-2.5 py-1.5 bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <option value="">Top-level project</option>
        {topLevel.map((p) => (
          <option key={p.id} value={p.id}>
            Inside: {p.name}
          </option>
        ))}
      </select>
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          checked={sequential}
          onChange={(e) => setSequential(e.target.checked)}
          className="rounded border-gray-300"
        />
        Sequential (only the first task is available)
      </label>
      <label className="flex items-center gap-2 text-sm text-gray-700">
        Review every
        <input
          type="number"
          min={1}
          value={interval}
          onChange={(e) => setInterval(Math.max(1, Number(e.target.value) || 1))}
          className="w-16 border border-gray-300 rounded-md px-2 py-1 text-sm"
        />
        days
      </label>
      <div className="flex items-center gap-2 pt-1">
        <button
          type="submit"
          disabled={busy || !name.trim()}
          className="text-sm font-medium px-3 py-1.5 rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
        >
          Create
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="text-sm px-3 py-1.5 rounded-md text-gray-600 hover:bg-gray-100"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
