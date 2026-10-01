"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Project } from "@/lib/types";
import { useToast } from "@/contexts/ToastContext";
import { parentOptions } from "@/hooks/useProjectsAndTags";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const TOP = "__top__";

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
  const [parentId, setParentId] = useState(defaultParentId ?? TOP);
  const [sequential, setSequential] = useState(false);
  const [interval, setInterval] = useState(7);
  const [busy, setBusy] = useState(false);
  // Buckets (depth 0) and projects (depth 1) can hold children; three levels max.
  const parents = parentOptions(projects);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/projects", {
      body: {
        name: name.trim(),
        parent_id: parentId === TOP ? undefined : parentId,
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
      <Input
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Project name"
        autoFocus
      />
      <Select value={parentId} onValueChange={setParentId}>
        <SelectTrigger aria-label="Parent project">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={TOP}>Top-level (bucket or project)</SelectItem>
          {parents.map(({ project: p, depth }) => (
            <SelectItem key={p.id} value={p.id} depth={depth + 1}>
              Inside {p.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <Checkbox
          checked={sequential}
          onCheckedChange={(c) => setSequential(c === true)}
        />
        Sequential (only the first task is available)
      </label>
      <label className="flex items-center gap-2 text-sm text-gray-700">
        Review every
        <Input
          type="number"
          min={1}
          value={interval}
          onChange={(e) =>
            setInterval(Math.max(1, Number(e.target.value) || 1))
          }
          className="w-16"
        />
        days
      </label>
      <div className="flex items-center gap-2 pt-1">
        <Button type="submit" variant="primary" disabled={busy || !name.trim()}>
          Create
        </Button>
        <Button variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
