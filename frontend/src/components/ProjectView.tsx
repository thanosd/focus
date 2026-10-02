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
import { subtree } from "@/lib/projects";
import { useCounts } from "@/contexts/CountsContext";
import { useConfirm } from "@/contexts/ConfirmContext";
import { useToast } from "@/contexts/ToastContext";
import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  ChevronLeft,
  ChevronRight,
  CornerDownRight,
  SlidersHorizontal,
} from "lucide-react";
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
  /** Every project, all statuses, for the hierarchy shown in the pane. */
  treeProjects?: Project[];
  tags: Tag[];
  onCreateTag: (name: string) => Promise<Tag | null>;
  /** Called after any project-level change so parents can refresh lists. */
  onProjectChanged: (project: Project) => void;
  onProjectDeleted?: (id: string) => void;
  /** Deselect the project (closes the pane on the projects page). */
  onClose?: () => void;
  /** Phone-only back control shown in the items column header. */
  backHref?: string;
  backLabel?: string;
  /** Extra buttons rendered next to "Mark reviewed" (review mode). */
  actions?: React.ReactNode;
  /** Review mode: tighter header. */
  compact?: boolean;
}

/**
 * Project page body. The items column is the project's whole outline: its
 * own quick-add and tasks, then every sub-project (depth-first) with its own
 * quick-add and tasks. Selecting a smaller sub-project narrows the view. The
 * properties column shows the project's properties, or the selected task's
 * inspector. Used by /projects/[id] and the review page.
 */
export default function ProjectView({
  projectId,
  projects,
  treeProjects,
  tags,
  onCreateTag,
  onProjectChanged,
  onProjectDeleted,
  onClose,
  backHref,
  backLabel = "Back",
  actions,
  compact = false,
}: ProjectViewProps) {
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const confirm = useConfirm();
  const [detail, setDetail] = useState<ProjectDetail | null>(null);
  // Every task in the subtree (this project + all descendants), keyed by
  // task.project_id when rendering the outline.
  const [tasks, setTasks] = useState<Task[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Task | null>(null);
  const [showDone, setShowDone] = useState(false);
  const [busy, setBusy] = useState(false);
  // Phone: the project's properties open as a sheet from this button.
  const [propsOpen, setPropsOpen] = useState(false);

  const loadTasks = useCallback(async () => {
    const { data, error } = await apiClient.GET("/api/tasks", {
      params: {
        query: {
          view: "all",
          project_id: projectId,
          include_subprojects: true,
        },
      },
    });
    if (error || !data) {
      toast(errorMessage(error, "Couldn't load tasks"), "error");
      return;
    }
    setTasks(data);
  }, [projectId, toast]);

  const load = useCallback(async () => {
    const [{ data, error }] = await Promise.all([
      apiClient.GET("/api/projects/{projectId}", {
        params: { path: { projectId } },
      }),
      loadTasks(),
    ]);
    if (error || !data) {
      setError(errorMessage(error, "Project not found"));
      return;
    }
    setError(null);
    setDetail(data);
  }, [projectId, loadTasks]);

  useEffect(() => {
    setDetail(null);
    setTasks([]);
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

  // Hierarchy for the outline (sub-projects and theirs), computed from the
  // full project list so it survives status filtering in the tree.
  const descendants = useMemo(
    () =>
      subtree(
        treeProjects && treeProjects.length > 0 ? treeProjects : projects,
        projectId,
      ),
    [treeProjects, projects, projectId],
  );
  const subtreeIds = useMemo(() => {
    const ids = new Set<string>([projectId]);
    for (const { project: p } of descendants) ids.add(p.id);
    return ids;
  }, [descendants, projectId]);

  const updateTask = (task: Task) => {
    setTasks((list) => {
      const belongs = !!task.project_id && subtreeIds.has(task.project_id);
      const exists = list.some((t) => t.id === task.id);
      if (belongs && exists)
        return list.map((t) => (t.id === task.id ? task : t));
      if (belongs) return [...list, task];
      return list.filter((t) => t.id !== task.id);
    });
    setSelected((s) => (s && s.id === task.id ? task : s));
  };

  /** Reorder within one project's group; ids are that group's active tasks. */
  const reorder = async (ids: string[]) => {
    const before = tasks;
    setTasks((list) => {
      const group = new Set(ids);
      const ordered = applyOrder(
        list.filter((t) => group.has(t.id)),
        ids,
      );
      let i = 0;
      return list.map((t) => (group.has(t.id) ? ordered[i++] : t));
    });
    const { data, error } = await apiClient.POST("/api/tasks/reorder", {
      body: { task_ids: ids },
    });
    if (error || !data) {
      setTasks(before);
      toast(errorMessage(error, "Couldn't reorder tasks"), "error");
      return;
    }
    // The response carries fresh availability (sequential projects shift
    // which task is first).
    const fresh = new Map(data.map((t) => [t.id, t]));
    setTasks((list) => list.map((t) => fresh.get(t.id) ?? t));
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
    setTasks((list) => list.filter((t) => t.id !== id));
    setSelected((s) => (s && s.id === id ? null : s));
    refreshProjectCounts();
  };

  const closeTask = useCallback(() => setSelected(null), []);
  const closePane = useCallback(() => {
    setSelected(null);
    if (propsOpen) setPropsOpen(false);
    else onClose?.();
  }, [onClose, propsOpen]);

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
  const hierarchy =
    descendants.length > 0
      ? descendants
      : children.map((c) => ({ project: c, depth: 1 }));
  const tasksOf = (id: string) =>
    tasks.filter(
      (t) => t.project_id === id && (showDone || t.status === "active"),
    );
  const doneCount = tasks.filter((t) => t.status !== "active").length;

  const listProps = {
    projects,
    hideProject: true,
    selectedId: selected?.id ?? null,
    onSelect: setSelected,
    onUpdated: (t: Task) => {
      updateTask(t);
      refreshProjectCounts();
    },
    onCompleted: completeTask,
    onDeleted: deleteTask,
  };

  return (
    // Columns: items | properties.
    <div className="flex flex-1 items-start min-w-0">
      <div className="flex-1 min-w-0 p-4 md:p-8">
        <div className="space-y-4">
          <div>
            {backHref && (
              <Link
                href={backHref}
                className="md:hidden inline-flex items-center gap-1 py-2 -ml-1 pr-2 text-sm text-blue-600"
              >
                <ChevronLeft className="w-4 h-4" />
                {backLabel}
              </Link>
            )}
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
                <Button
                  size="xs"
                  className="md:hidden min-h-[40px]"
                  onClick={() => {
                    setSelected(null);
                    setPropsOpen(true);
                  }}
                  aria-label="Project properties"
                >
                  <SlidersHorizontal className="w-3.5 h-3.5" /> Properties
                </Button>
              </div>
            </div>
          </div>

          <QuickAdd
            projectId={projectId}
            placeholder={`Add a task to ${project.name}…`}
            onCreated={(t) => {
              updateTask(t);
              refreshProjectCounts();
            }}
          />
          <TaskList
            tasks={tasksOf(projectId)}
            emptyMessage={
              hierarchy.length > 0
                ? "No tasks of its own."
                : project.status === "active"
                  ? "No tasks. Add one above."
                  : "No active tasks."
            }
            numbered={project.sequential}
            onReorder={reorder}
            {...listProps}
          />

          {/* Outline: every sub-project (depth-first) with its own tasks.
              Selecting a smaller sub-project narrows the view. */}
          {hierarchy.map(({ project: sub, depth }) => (
            <section
              key={sub.id}
              className="space-y-2"
              style={{ marginLeft: `${(depth - 1) * 20}px` }}
            >
              <div className="flex items-center gap-2 pt-2 min-w-0">
                {depth > 1 && (
                  <CornerDownRight className="w-3.5 h-3.5 text-gray-300 flex-shrink-0" />
                )}
                <Link
                  href={`/projects/${sub.id}`}
                  className="text-sm font-semibold text-gray-800 hover:text-blue-600 truncate"
                >
                  {sub.name}
                </Link>
                <StatusBadge status={sub.status} />
                <span
                  className="text-[11px] text-gray-400 tabular-nums flex-shrink-0"
                  title={`${sub.available_task_count} available / ${sub.remaining_task_count} remaining`}
                >
                  {sub.available_task_count}/{sub.remaining_task_count}
                </span>
              </div>
              <QuickAdd
                projectId={sub.id}
                placeholder={`Add a task to ${sub.name}…`}
                onCreated={(t) => {
                  updateTask(t);
                  refreshProjectCounts();
                }}
              />
              <TaskList
                tasks={tasksOf(sub.id)}
                emptyMessage={
                  sub.status === "active" ? "No tasks yet." : "No active tasks."
                }
                numbered={sub.sequential}
                onReorder={reorder}
                {...listProps}
              />
            </section>
          ))}

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
        open={!!selected || propsOpen}
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
