"use client";

import type { Project } from "@/lib/types";
import { projectOptions } from "@/hooks/useProjectsAndTags";

interface ProjectPickerProps {
  projects: Project[];
  value: string | undefined;
  onChange: (projectId: string | null) => void;
  disabled?: boolean;
  className?: string;
  /** Compact variant used inside task rows. */
  compact?: boolean;
}

/** <select> of projects with nested projects indented under their parent. */
export default function ProjectPicker({
  projects,
  value,
  onChange,
  disabled,
  className = "",
  compact = false,
}: ProjectPickerProps) {
  const options = projectOptions(
    projects.filter((p) => p.status === "active" || p.status === "on_hold" || p.id === value),
  );
  return (
    <select
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value || null)}
      disabled={disabled}
      onClick={(e) => e.stopPropagation()}
      className={`border border-gray-300 rounded-md bg-white text-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500 ${
        compact ? "text-xs px-1.5 py-0.5" : "text-sm px-2.5 py-1.5"
      } ${className}`}
    >
      <option value="">Inbox (no project)</option>
      {options.map(({ project, depth }) => (
        <option key={project.id} value={project.id}>
          {depth > 0 ? "    ↳ " : ""}
          {project.name}
          {project.status !== "active" ? ` (${project.status.replace("_", " ")})` : ""}
        </option>
      ))}
    </select>
  );
}
