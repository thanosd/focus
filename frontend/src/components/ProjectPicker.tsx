"use client";

import type { Project } from "@/lib/types";
import { projectOptions } from "@/hooks/useProjectsAndTags";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Inbox } from "lucide-react";
import { cn } from "@/lib/utils";

const INBOX = "__inbox__";

interface ProjectPickerProps {
  projects: Project[];
  value: string | undefined;
  onChange: (projectId: string | null) => void;
  disabled?: boolean;
  className?: string;
  /** Compact variant used inside task rows. */
  compact?: boolean;
}

/** Project dropdown with nested projects indented under their parent. */
export default function ProjectPicker({
  projects,
  value,
  onChange,
  disabled,
  className = "",
  compact = false,
}: ProjectPickerProps) {
  const options = projectOptions(
    projects.filter(
      (p) => p.status === "active" || p.status === "on_hold" || p.id === value,
    ),
  );
  return (
    <Select
      value={value ?? INBOX}
      onValueChange={(v) => onChange(v === INBOX ? null : v)}
      disabled={disabled}
    >
      <SelectTrigger
        size={compact ? "xs" : "sm"}
        className={cn(compact ? "w-44" : "w-full", className)}
        onClick={(e) => e.stopPropagation()}
        aria-label="Project"
      >
        <SelectValue placeholder="Project" />
      </SelectTrigger>
      <SelectContent onClick={(e) => e.stopPropagation()}>
        <SelectItem value={INBOX}>
          <span className="inline-flex items-center gap-1.5 text-gray-600">
            <Inbox className="h-3.5 w-3.5" /> Inbox (no project)
          </span>
        </SelectItem>
        {options.length > 0 && <SelectSeparator />}
        {options.map(({ project, depth }) => (
          <SelectItem key={project.id} value={project.id} depth={depth}>
            {project.name}
            {project.status !== "active" && (
              <span className="ml-1.5 text-xs text-gray-400">
                ({project.status.replace("_", " ")})
              </span>
            )}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
