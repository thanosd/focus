"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type {
  Project,
  ProjectDetail,
  ProjectStatus,
  Tag,
  Task,
  UpdateProjectRequest,
} from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useCounts } from "@/contexts/CountsContext";
import { useToast } from "@/contexts/ToastContext";
import { dayLabel } from "@/lib/dates";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import TaskList from "@/components/TaskList";
import TaskInspector from "@/components/TaskInspector";
import QuickAdd from "@/components/QuickAdd";
import { StatusBadge } from "@/components/ProjectTree";

interface ProjectViewProps {
  projectId: string;
  projects: Project[];
  tags: Tag[];
  onCreateTag: (name: string) => Promise<Tag | null>;
  /** Called after any project-level change so parents can refresh lists. */
  onProjectChanged: (project: Project) => void;
  onProjectDeleted?: (id: string) => void;
  /** Extra buttons rendered next to "Mark reviewed" (review mode). */
  actions?: React.ReactNode;
  /** Show completed/dropped tasks too. */
  compact?: boolean;
}

/**
 * Full project panel: editable header, child projects, task list with
 * quick-add and inspector. Used by /projects/[id] and the review page.
 */
export default function ProjectView({
  projectId,
  projects,
  tags,
  onCreateTag,
  onProjectChanged,
  onProjectDeleted,
  actions,
  compact = false,
}: ProjectViewProps) {
  const { timezone } = useAuth();
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const [detail, setDetail] = useState<ProjectDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Task | null>(null);
  const [name, setName] = useState("");
  const [note, setNote] = useState("");
  const [showDone, setShowDone] = useState(false);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const { data, error } = await apiClient.GET("/api/projects/{projectId}", {
      params: { path: { projectId } },
    });
    if (error || !data) {
      setError(errorMessage(error, "Project not found"));
      return;
    }
    setError(null);
    setDetail(data);
    setName(data.project.name);
    setNote(data.project.note);
  }, [projectId]);

  useEffect(() => {
    setDetail(null);
    setSelected(null);
    load();
  }, [load]);

  const patchProject = async (body: UpdateProjectRequest) => {
    if (!detail) return;
    setBusy(true);
    const { data, error } = await apiClient.PATCH("/api/projects/{projectId}", {
      params: { path: { projectId } },
      body,
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't update project"), "error");
      return;
    }
    setDetail({ ...detail, project: data });
    onProjectChanged(data);
    refreshCounts();
  };

  const review = async () => {
    setBusy(true);
    const { data, error } = await apiClient.POST(
      "/api/projects/{projectId}/review",
      { params: { path: { projectId } } },
    );
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't mark reviewed"), "error");
      return;
    }
    if (detail) setDetail({ ...detail, project: data });
    onProjectChanged(data);
    refreshCounts();
    toast("Project reviewed", "success");
  };

  const del = async () => {
    if (!detail) return;
    const n = detail.children.length;
    if (
      !window.confirm(
        `Delete "${detail.project.name}"${n ? ` and its ${n} sub-project${n > 1 ? "s" : ""}` : ""} with all tasks?`,
      )
    )
      return;
    const { error } = await apiClient.DELETE("/api/projects/{projectId}", {
      params: { path: { projectId } },
    });
    if (error) {
      toast(errorMessage(error, "Couldn't delete project"), "error");
      return;
    }
    refreshCounts();
    onProjectDeleted?.(projectId);
  };

  const updateTask = (task: Task) => {
    setDetail((d) => {
      if (!d) return d;
      const belongs = task.project_id === projectId;
      const exists = d.tasks.some((t) => t.id === task.id);
      let tasks = d.tasks;
      if (belongs && exists)
        tasks = d.tasks.map((t) => (t.id === task.id ? task : t));
      else if (belongs) tasks = [task, ...d.tasks];
      else tasks = d.tasks.filter((t) => t.id !== task.id);
      return { ...d, tasks };
    });
    setSelected((s) => (s && s.id === task.id ? task : s));
  };

  const completeTask = (task: Task, next?: Task) => {
    updateTask(task);
    if (next) updateTask(next);
    // Counts on the project header change; refresh quietly.
    refreshProjectCounts();
  };

  const refreshProjectCounts = async () => {
    const { data } = await apiClient.GET("/api/projects/{projectId}", {
      params: { path: { projectId } },
    });
    if (data) {
      setDetail((d) =>
        d ? { ...d, project: data.project, children: data.children } : d,
      );
      onProjectChanged(data.project);
    }
  };

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
        {error}
      </div>
    );
  }
  if (!detail) {
    return <div className="text-sm text-gray-500 p-4">Loading…</div>;
  }

  const { project, children } = detail;
  const parent = project.parent_id
    ? projects.find((p) => p.id === project.parent_id)
    : undefined;
  const visibleTasks = showDone
    ? detail.tasks
    : detail.tasks.filter((t) => t.status === "active");
  const doneCount =
    detail.tasks.length -
    detail.tasks.filter((t) => t.status === "active").length;

  return (
    <div className="space-y-4">
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        {parent && (
          <Link
            href={`/projects/${parent.id}`}
            className="text-xs text-gray-500 hover:text-blue-600"
          >
            ↑ {parent.name}
          </Link>
        )}
        <div className="flex items-start gap-2">
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onBlur={() => {
              const n = name.trim();
              if (!n) setName(project.name);
              else if (n !== project.name) patchProject({ name: n });
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") (e.target as HTMLInputElement).blur();
            }}
            disabled={busy}
            className="flex-1 text-xl font-bold text-gray-900 border-b border-transparent hover:border-gray-200 focus:border-blue-500 focus:outline-none bg-transparent"
          />
          <StatusBadge status={project.status} />
        </div>
        <textarea
          value={note}
          onChange={(e) => setNote(e.target.value)}
          onBlur={() => {
            if (note !== project.note) patchProject({ note });
          }}
          disabled={busy}
          rows={compact ? 2 : 3}
          placeholder="Project note — what does done look like?"
          className="w-full text-sm border border-gray-200 rounded-md px-2.5 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-gray-700">
          <label className="flex items-center gap-1.5">
            Status
            <select
              value={project.status}
              disabled={busy}
              onChange={(e) =>
                patchProject({ status: e.target.value as ProjectStatus })
              }
              className="border border-gray-300 rounded-md px-2 py-1 text-sm bg-white"
            >
              <option value="active">Active</option>
              <option value="on_hold">On hold</option>
              <option value="completed">Completed</option>
              <option value="dropped">Dropped</option>
            </select>
          </label>
          <label className="flex items-center gap-1.5">
            <input
              type="checkbox"
              checked={project.sequential}
              disabled={busy}
              onChange={(e) => patchProject({ sequential: e.target.checked })}
              className="rounded border-gray-300"
            />
            Sequential
          </label>
          <label className="flex items-center gap-1.5">
            Review every
            <input
              type="number"
              min={1}
              defaultValue={project.review_interval_days}
              key={project.review_interval_days}
              disabled={busy}
              onBlur={(e) => {
                const v = Math.max(1, Number(e.target.value) || 1);
                if (v !== project.review_interval_days)
                  patchProject({ review_interval_days: v });
              }}
              className="w-14 border border-gray-300 rounded-md px-2 py-1 text-sm"
            />
            days
          </label>
          {!project.parent_id && (
            <label className="flex items-center gap-1.5">
              Parent
              <select
                value=""
                disabled={busy || children.length > 0}
                title={
                  children.length > 0
                    ? "Projects with sub-projects must stay top-level"
                    : undefined
                }
                onChange={(e) => {
                  if (e.target.value)
                    patchProject({ parent_id: e.target.value });
                }}
                className="border border-gray-300 rounded-md px-2 py-1 text-sm bg-white"
              >
                <option value="">Top-level</option>
                {projects
                  .filter(
                    (p) =>
                      !p.parent_id &&
                      p.id !== project.id &&
                      p.status === "active",
                  )
                  .map((p) => (
                    <option key={p.id} value={p.id}>
                      Move under {p.name}
                    </option>
                  ))}
              </select>
            </label>
          )}
          {project.parent_id && (
            <button
              type="button"
              disabled={busy}
              onClick={() => patchProject({ parent_id: null })}
              className="text-xs text-gray-500 hover:text-blue-600"
            >
              Move to top level
            </button>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2 pt-1 border-t border-gray-100">
          <button
            type="button"
            onClick={review}
            disabled={busy}
            className="text-sm font-medium px-3 py-1.5 rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
          >
            Mark reviewed
          </button>
          {actions}
          <span className="text-xs text-gray-500">
            {project.last_reviewed_at
              ? `Last reviewed ${dayLabel(project.last_reviewed_at, timezone)}`
              : "Never reviewed"}
            {project.next_review_at
              ? ` · next ${dayLabel(project.next_review_at, timezone)}`
              : ""}
          </span>
          {onProjectDeleted && (
            <button
              type="button"
              onClick={del}
              disabled={busy}
              className="ml-auto text-xs text-red-500 hover:text-red-700"
            >
              Delete project
            </button>
          )}
        </div>
      </div>

      {children.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-lg">
          <div className="px-4 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wide border-b border-gray-100">
            Sub-projects
          </div>
          {children.map((c) => (
            <Link
              key={c.id}
              href={`/projects/${c.id}`}
              className="flex items-center gap-2 px-4 py-2 text-sm text-gray-800 hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
            >
              <span className="flex-1 truncate">{c.name}</span>
              <StatusBadge status={c.status} />
              <span className="text-[11px] text-gray-400 tabular-nums">
                {c.available_task_count}/{c.remaining_task_count}
              </span>
            </Link>
          ))}
        </div>
      )}

      <QuickAdd
        projectId={projectId}
        placeholder={`Add a task to ${project.name}…`}
        onCreated={(t) => {
          updateTask(t);
          setSelected(t);
          refreshProjectCounts();
        }}
      />
      <TaskList
        tasks={visibleTasks}
        emptyMessage={
          project.status === "active"
            ? "No tasks. Add one above."
            : "No active tasks."
        }
        projects={projects}
        hideProject
        selectedId={selected?.id ?? null}
        onSelect={setSelected}
        onUpdated={(t) => {
          updateTask(t);
          refreshProjectCounts();
        }}
        onCompleted={completeTask}
      />
      {doneCount > 0 && (
        <button
          type="button"
          onClick={() => setShowDone((v) => !v)}
          className="text-xs text-gray-500 hover:text-gray-800"
        >
          {showDone ? "Hide" : "Show"} {doneCount} completed/dropped
        </button>
      )}

      {selected && (
        <TaskInspector
          task={selected}
          projects={projects}
          tags={tags}
          onCreateTag={onCreateTag}
          onUpdated={(t) => {
            updateTask(t);
            refreshProjectCounts();
          }}
          onCompleted={completeTask}
          onDeleted={(id) => {
            setDetail((d) =>
              d ? { ...d, tasks: d.tasks.filter((t) => t.id !== id) } : d,
            );
            setSelected(null);
            refreshProjectCounts();
          }}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}
