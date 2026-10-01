"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type {
  Project,
  ProjectDetail,
  Tag,
  Task,
  UpdateProjectRequest,
} from "@/lib/types";
import { applyOrder } from "@/lib/availability";
import { useCounts } from "@/contexts/CountsContext";
import { useConfirm } from "@/contexts/ConfirmContext";
import { useToast } from "@/contexts/ToastContext";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { ChevronRight } from "lucide-react";
import TaskList from "@/components/TaskList";
import TaskInspector from "@/components/TaskInspector";
import ProjectInspector from "@/components/ProjectInspector";
import QuickAdd from "@/components/QuickAdd";
import DockedPane from "@/components/DockedPane";
import { StatusBadge } from "@/components/ProjectTree";
import { Button } from "@/components/ui/button";

interface ProjectViewProps {
  projectId: string;
  projects: Project[];
  tags: Tag[];
  onCreateTag: (name: string) => Promise<Tag | null>;
  /** Called after any project-level change so parents can refresh lists. */
  onProjectChanged: (project: Project) => void;
  onProjectDeleted?: (id: string) => void;
  /** Deselect the project (closes the pane on the projects page). */
  onClose?: () => void;
  /** Extra buttons rendered next to "Mark reviewed" (review mode). */
  actions?: React.ReactNode;
  /** Review mode: tighter header. */
  compact?: boolean;
}

/**
 * Project page body: sub-projects, quick-add and task list in the main
 * column; the docked pane shows the project's properties, or the selected
 * task's inspector. Used by /projects/[id] and the review page.
 */
export default function ProjectView({
  projectId,
  projects,
  tags,
  onCreateTag,
  onProjectChanged,
  onProjectDeleted,
  onClose,
  actions,
  compact = false,
}: ProjectViewProps) {
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const confirm = useConfirm();
  const [detail, setDetail] = useState<ProjectDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Task | null>(null);
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
    const ok = await confirm({
      title: `Delete "${detail.project.name}"?`,
      description: `${
        n ? `Its ${n} sub-project${n > 1 ? "s" : ""} and all` : "All"
      } of its tasks will be permanently deleted.`,
      confirmLabel: "Delete project",
      destructive: true,
    });
    if (!ok) return;
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
      else if (belongs) tasks = [...d.tasks, task];
      else tasks = d.tasks.filter((t) => t.id !== task.id);
      return { ...d, tasks };
    });
    setSelected((s) => (s && s.id === task.id ? task : s));
  };

  const reorder = async (ids: string[]) => {
    if (!detail) return;
    const before = detail.tasks;
    setDetail((d) => (d ? { ...d, tasks: applyOrder(d.tasks, ids) } : d));
    const { data, error } = await apiClient.POST("/api/tasks/reorder", {
      body: { task_ids: ids },
    });
    if (error || !data) {
      setDetail((d) => (d ? { ...d, tasks: before } : d));
      toast(errorMessage(error, "Couldn't reorder tasks"), "error");
      return;
    }
    // The response carries fresh availability (sequential projects shift
    // which task is first).
    const fresh = new Map(data.map((t) => [t.id, t]));
    setDetail((d) =>
      d ? { ...d, tasks: d.tasks.map((t) => fresh.get(t.id) ?? t) } : d,
    );
    setSelected((s) => (s && fresh.get(s.id)) || s);
    refreshProjectCounts();
  };

  const refreshProjectCounts = useCallback(async () => {
    const { data } = await apiClient.GET("/api/projects/{projectId}", {
      params: { path: { projectId } },
    });
    if (data) {
      setDetail((d) =>
        d ? { ...d, project: data.project, children: data.children } : d,
      );
      onProjectChanged(data.project);
    }
  }, [projectId, onProjectChanged]);

  const completeTask = (task: Task, next?: Task) => {
    updateTask(task);
    if (next) updateTask(next);
    refreshProjectCounts();
  };

  const deleteTask = (id: string) => {
    setDetail((d) =>
      d ? { ...d, tasks: d.tasks.filter((t) => t.id !== id) } : d,
    );
    setSelected((s) => (s && s.id === id ? null : s));
    refreshProjectCounts();
  };

  const closeTask = useCallback(() => setSelected(null), []);
  const closePane = useCallback(() => {
    setSelected(null);
    onClose?.();
  }, [onClose]);

  if (error) {
    return (
      <div className="p-6 md:p-8 flex-1">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
        </div>
      </div>
    );
  }
  if (!detail) {
    return (
      <div className="p-6 md:p-8 text-sm text-gray-500 flex-1">Loading…</div>
    );
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
    <div className="flex flex-1 items-start">
      <div className="flex-1 min-w-0 p-6 md:p-8">
        <div className="max-w-4xl space-y-4">
          <div>
            {parent && (
              <Link
                href={`/projects/${parent.id}`}
                className="inline-flex items-center gap-1 text-xs text-gray-500 hover:text-blue-600"
              >
                {parent.name}
                <ChevronRight className="w-3 h-3" />
              </Link>
            )}
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <h1 className="text-2xl font-bold text-gray-900 truncate">
                  {project.name}
                </h1>
                {!compact && project.note && (
                  <p className="text-sm text-gray-500 mt-0.5 line-clamp-2 whitespace-pre-line">
                    {project.note}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-2 flex-shrink-0 pt-1">
                <StatusBadge status={project.status} />
              </div>
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
            numbered={project.sequential}
            onReorder={reorder}
            selectedId={selected?.id ?? null}
            onSelect={setSelected}
            onUpdated={(t) => {
              updateTask(t);
              refreshProjectCounts();
            }}
            onCompleted={completeTask}
            onDeleted={deleteTask}
          />
          {doneCount > 0 && (
            <Button
              variant="ghost"
              size="xs"
              onClick={() => setShowDone((v) => !v)}
            >
              {showDone ? "Hide" : "Show"} {doneCount} completed/dropped
            </Button>
          )}
        </div>
      </div>

      <DockedPane
        open={!!selected}
        desktopAlwaysOpen
        onClose={closePane}
        label={selected ? "Task details" : "Project details"}
      >
        {selected ? (
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
            onDeleted={deleteTask}
            onClose={closeTask}
          />
        ) : (
          <ProjectInspector
            project={project}
            projects={projects}
            childCount={children.length}
            busy={busy}
            onPatch={patchProject}
            onReview={review}
            onDelete={onProjectDeleted ? del : undefined}
            actions={actions}
            onClose={closePane}
          />
        )}
      </DockedPane>
    </div>
  );
}
