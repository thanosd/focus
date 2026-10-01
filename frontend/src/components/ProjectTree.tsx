"use client";

import type { Project } from "@/lib/types";
import Link from "next/link";
import { useState } from "react";
import { ChevronRight } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";

interface ProjectTreeProps {
  projects: Project[];
  selectedId?: string | null;
  /** Show on_hold / completed / dropped projects too. */
  showInactive: boolean;
  onToggleInactive: (v: boolean) => void;
}

export function StatusBadge({ status }: { status: Project["status"] }) {
  if (status === "active") return null;
  const cls = {
    on_hold: "bg-amber-50 text-amber-700",
    completed: "bg-blue-50 text-blue-700",
    dropped: "bg-gray-100 text-gray-500",
  }[status];
  return (
    <span
      className={cn(
        "text-[10px] uppercase tracking-wide font-semibold px-1.5 py-0.5 rounded",
        cls,
      )}
    >
      {status.replace("_", " ")}
    </span>
  );
}

/** Two-level tree of projects with task counts. */
export default function ProjectTree({
  projects,
  selectedId,
  showInactive,
  onToggleInactive,
}: ProjectTreeProps) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const visible = showInactive
    ? projects
    : projects.filter((p) => p.status === "active" || p.id === selectedId);
  const sortFn = (a: Project, b: Project) =>
    a.sort_order - b.sort_order || a.name.localeCompare(b.name);
  const top = visible.filter((p) => !p.parent_id).sort(sortFn);
  const childrenOf = (id: string) =>
    visible.filter((p) => p.parent_id === id).sort(sortFn);
  // Children whose parent is hidden by the filter still need a home.
  const orphans = visible
    .filter((p) => p.parent_id && !visible.some((x) => x.id === p.parent_id))
    .sort(sortFn);

  const row = (p: Project, depth: number) => {
    const active = p.id === selectedId;
    const kids = depth === 0 ? childrenOf(p.id) : [];
    const isCollapsed = collapsed.has(p.id);
    return (
      <div key={p.id}>
        <Link
          href={`/projects/${p.id}`}
          className={cn(
            "flex items-center gap-2 py-1.5 pr-2 rounded-md text-sm transition-colors",
            active
              ? "bg-blue-50 text-blue-700 font-medium"
              : "text-gray-700 hover:bg-gray-100",
          )}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
        >
          {depth === 0 ? (
            <button
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setCollapsed((s) => {
                  const n = new Set(s);
                  if (n.has(p.id)) n.delete(p.id);
                  else n.add(p.id);
                  return n;
                });
              }}
              className={cn(
                "w-4 h-4 flex items-center justify-center text-gray-400 rounded hover:text-gray-600",
                kids.length === 0 && "invisible",
              )}
              aria-label={isCollapsed ? "Expand" : "Collapse"}
            >
              <ChevronRight
                className={cn(
                  "w-3.5 h-3.5 transition-transform",
                  !isCollapsed && "rotate-90",
                )}
              />
            </button>
          ) : (
            <span className="w-4 h-4" />
          )}
          <span className="truncate flex-1">{p.name}</span>
          <StatusBadge status={p.status} />
          <span
            className="text-[11px] text-gray-400 tabular-nums"
            title={`${p.available_task_count} available / ${p.remaining_task_count} remaining`}
          >
            {p.available_task_count}/{p.remaining_task_count}
          </span>
        </Link>
        {!isCollapsed && kids.map((k) => row(k, 1))}
      </div>
    );
  };

  return (
    <div>
      <div className="space-y-0.5">
        {top.map((p) => row(p, 0))}
        {orphans.map((p) => row(p, 0))}
        {top.length === 0 && orphans.length === 0 && (
          <div className="text-sm text-gray-500 px-2 py-4">
            No projects yet.
          </div>
        )}
      </div>
      <label className="mt-3 flex items-center gap-2 text-xs text-gray-500 px-2">
        <Checkbox
          checked={showInactive}
          onCheckedChange={(c) => onToggleInactive(c === true)}
        />
        Show on hold, completed and dropped
      </label>
    </div>
  );
}
