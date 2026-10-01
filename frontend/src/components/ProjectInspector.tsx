"use client";

import type { Project, ProjectStatus, UpdateProjectRequest } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useConfirm } from "@/contexts/ConfirmContext";
import { dayLabel } from "@/lib/dates";
import { useEffect, useState } from "react";
import {
  CheckCircle2,
  PauseCircle,
  PlayCircle,
  X,
  XCircle,
} from "lucide-react";
import { Field } from "@/components/TaskInspector";
import { StatusBadge } from "@/components/ProjectTree";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const TOP = "__top__";

interface ProjectInspectorProps {
  project: Project;
  /** All candidate parents (top-level, active). */
  projects: Project[];
  childCount: number;
  busy?: boolean;
  onPatch: (body: UpdateProjectRequest) => void;
  onReview: () => void;
  onDelete?: () => void;
  /** Extra actions (review mode). */
  actions?: React.ReactNode;
  onClose: () => void;
}

/** Project properties, shown in the docked pane. */
export default function ProjectInspector({
  project,
  projects,
  childCount,
  busy = false,
  onPatch,
  onReview,
  onDelete,
  actions,
  onClose,
}: ProjectInspectorProps) {
  const { timezone } = useAuth();
  const confirm = useConfirm();
  const [name, setName] = useState(project.name);
  const [note, setNote] = useState(project.note);

  useEffect(() => {
    setName(project.name);
    setNote(project.note);
  }, [project.id, project.name, project.note]);

  const parents = projects.filter(
    (p) => !p.parent_id && p.id !== project.id && p.status === "active",
  );
  const canNest = childCount === 0;
  const closed = project.status === "completed" || project.status === "dropped";

  const setStatus = async (status: ProjectStatus) => {
    if (status === "completed" || status === "dropped") {
      const n = project.remaining_task_count;
      const ok = await confirm({
        title: `${status === "completed" ? "Complete" : "Drop"} "${project.name}"?`,
        description:
          n > 0
            ? `${n} remaining task${n === 1 ? "" : "s"} will stop being available. The project stays in the archive and can be reactivated.`
            : "The project stays in the archive and can be reactivated.",
        confirmLabel:
          status === "completed" ? "Complete project" : "Drop project",
        destructive: status === "dropped",
      });
      if (!ok) return;
    }
    onPatch({ status });
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
        <div className="flex items-center gap-2">
          <span className="text-[10px] uppercase tracking-wide font-semibold px-1.5 py-0.5 rounded bg-gray-100 text-gray-600">
            project
          </span>
          <StatusBadge status={project.status} />
        </div>
        <Button
          size="icon"
          variant="ghost"
          onClick={onClose}
          aria-label="Close"
          title="Close (Esc)"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-5">
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={() => {
            const n = name.trim();
            if (!n) setName(project.name);
            else if (n !== project.name) onPatch({ name: n });
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") (e.target as HTMLInputElement).blur();
          }}
          disabled={busy}
          aria-label="Project name"
          className="w-full text-base font-medium text-gray-900 border-b border-transparent hover:border-gray-200 focus:border-blue-500 focus:outline-none py-1 bg-transparent"
        />

        <Field label="Note">
          <Textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            onBlur={() => {
              if (note !== project.note) onPatch({ note });
            }}
            disabled={busy}
            rows={4}
            placeholder="What does done look like?"
          />
        </Field>

        <Field label="Status">
          <Select
            value={project.status}
            disabled={busy}
            onValueChange={(v) => onPatch({ status: v as ProjectStatus })}
          >
            <SelectTrigger aria-label="Status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="on_hold">On hold</SelectItem>
              <SelectItem value="completed">Completed</SelectItem>
              <SelectItem value="dropped">Dropped</SelectItem>
            </SelectContent>
          </Select>
        </Field>

        <Field
          label="Sequential"
          hint="Only the first remaining task is available"
        >
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <Switch
              checked={project.sequential}
              disabled={busy}
              onCheckedChange={(c) => onPatch({ sequential: c })}
            />
            {project.sequential ? "On" : "Off"}
          </label>
        </Field>

        <Field label="Review">
          <div className="flex items-center gap-2 text-sm text-gray-700">
            every
            <Input
              type="number"
              min={1}
              key={project.review_interval_days}
              defaultValue={project.review_interval_days}
              disabled={busy}
              onBlur={(e) => {
                const v = Math.max(1, Number(e.target.value) || 1);
                if (v !== project.review_interval_days)
                  onPatch({ review_interval_days: v });
              }}
              className="w-16"
            />
            days
          </div>
          <p className="mt-1.5 text-xs text-gray-500">
            {project.last_reviewed_at
              ? `Last reviewed ${dayLabel(project.last_reviewed_at, timezone)}`
              : "Never reviewed"}
            {project.next_review_at
              ? ` · next ${dayLabel(project.next_review_at, timezone)}`
              : ""}
          </p>
        </Field>

        <Field
          label="Parent"
          hint={!canNest ? "Has sub-projects; must stay top-level" : undefined}
        >
          <Select
            value={project.parent_id ?? TOP}
            disabled={busy || (!canNest && !project.parent_id)}
            onValueChange={(v) => onPatch({ parent_id: v === TOP ? null : v })}
          >
            <SelectTrigger aria-label="Parent project">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={TOP}>Top-level</SelectItem>
              {parents.map((p) => (
                <SelectItem key={p.id} value={p.id} depth={1}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>

        <div className="text-xs text-gray-400 space-y-0.5 pt-2 border-t border-gray-100">
          <div>
            {project.available_task_count} available ·{" "}
            {project.remaining_task_count} remaining
          </div>
          <div>Created {dayLabel(project.created_at, timezone)}</div>
          {project.completed_at && (
            <div>Closed {dayLabel(project.completed_at, timezone)}</div>
          )}
        </div>

        <div className="pt-4 border-t border-gray-200 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="primary" onClick={onReview} disabled={busy}>
              Mark reviewed
            </Button>
            {actions}
          </div>
          {/* Review mode supplies its own status actions above. */}
          <div
            className={cn(
              "flex flex-wrap items-center gap-2",
              actions && "hidden",
            )}
          >
            {closed ? (
              <Button
                size="xs"
                onClick={() => setStatus("active")}
                disabled={busy}
              >
                <PlayCircle className="w-3.5 h-3.5" /> Reactivate
              </Button>
            ) : (
              <>
                <Button
                  size="xs"
                  onClick={() => setStatus("completed")}
                  disabled={busy}
                  title="Mark the whole project done"
                >
                  <CheckCircle2 className="w-3.5 h-3.5" /> Complete project
                </Button>
                {project.status === "on_hold" ? (
                  <Button
                    size="xs"
                    onClick={() => setStatus("active")}
                    disabled={busy}
                  >
                    <PlayCircle className="w-3.5 h-3.5" /> Reactivate
                  </Button>
                ) : (
                  <Button
                    size="xs"
                    onClick={() => setStatus("on_hold")}
                    disabled={busy}
                  >
                    <PauseCircle className="w-3.5 h-3.5" /> Put on hold
                  </Button>
                )}
                <Button
                  size="xs"
                  variant="danger-ghost"
                  onClick={() => setStatus("dropped")}
                  disabled={busy}
                >
                  <XCircle className="w-3.5 h-3.5" /> Drop project
                </Button>
              </>
            )}
            {onDelete && (
              <Button
                size="xs"
                variant="danger-ghost"
                onClick={onDelete}
                disabled={busy}
                className="ml-auto"
              >
                Delete
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
