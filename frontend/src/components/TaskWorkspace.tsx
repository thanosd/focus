"use client";

import type { Task } from "@/lib/types";
import { apiClient, errorMessage } from "@/lib/api-client";
import { applyOrder } from "@/lib/availability";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { useTasks, type TaskQuery } from "@/hooks/useTasks";
import { useToast } from "@/contexts/ToastContext";
import { useCallback, useState } from "react";
import TaskList from "@/components/TaskList";
import TaskInspector from "@/components/TaskInspector";
import QuickAdd from "@/components/QuickAdd";
import DockedPane from "@/components/DockedPane";
import BackLink from "@/components/BackLink";

interface TaskWorkspaceProps {
  title: string;
  subtitle?: string;
  query: TaskQuery;
  /** QuickAdd defaults: where created tasks go. */
  quickAdd?: { projectId?: string; tagIds?: string[]; flagged?: boolean };
  emptyMessage?: string;
  showProjectPicker?: boolean;
  hideProject?: boolean;
  /** Decide whether an updated task still belongs in this list. */
  keep?: (task: Task) => boolean;
  headerExtra?: React.ReactNode;
  /** Allow drag-and-drop reordering (inbox). */
  sortable?: boolean;
  /** Phone-only back control in the header. */
  backHref?: string;
  backLabel?: string;
}

/**
 * A task list page: heading, quick-add and list in the main column, with
 * the inspector docked on the right. Used by inbox, flagged and tag views.
 */
export default function TaskWorkspace({
  title,
  subtitle,
  query,
  quickAdd,
  emptyMessage,
  showProjectPicker,
  hideProject,
  keep,
  headerExtra,
  sortable = false,
  backHref,
  backLabel = "Back",
}: TaskWorkspaceProps) {
  const { tasks, loading, error, upsert, remove, append, setTasks } =
    useTasks(query);
  const { projects, tags, addTag } = useProjectsAndTags();
  const { toast } = useToast();
  const [selected, setSelected] = useState<Task | null>(null);

  const reorder = useCallback(
    async (ids: string[]) => {
      const before = tasks;
      setTasks((list) => applyOrder(list, ids));
      const { data, error } = await apiClient.POST("/api/tasks/reorder", {
        body: { task_ids: ids },
      });
      if (error || !data) {
        setTasks(before);
        toast(errorMessage(error, "Couldn't reorder tasks"), "error");
        return;
      }
      const fresh = new Map(data.map((t) => [t.id, t]));
      setTasks((list) => list.map((t) => fresh.get(t.id) ?? t));
    },
    [tasks, setTasks, toast],
  );

  const handleUpdated = useCallback(
    (task: Task) => {
      const stays = keep ? keep(task) : true;
      upsert(task, stays);
      setSelected((s) => (s && s.id === task.id ? task : s));
    },
    [keep, upsert],
  );

  const handleCompleted = useCallback(
    (task: Task, next?: Task) => {
      // Completed tasks leave active views; the inspector keeps showing it so
      // the user can reopen it if the click was a mistake.
      upsert(task, query.view === "completed" || query.view === "all");
      setSelected((s) => (s && s.id === task.id ? task : s));
      if (next && (keep ? keep(next) : true)) append(next);
    },
    [keep, append, upsert, query.view],
  );

  const handleDeleted = useCallback(
    (id: string) => {
      remove(id);
      setSelected((s) => (s && s.id === id ? null : s));
    },
    [remove],
  );

  const close = useCallback(() => setSelected(null), []);

  return (
    // Columns: items | properties.
    <div className="flex flex-1 items-start min-w-0">
      <div className="flex-1 min-w-0 p-4 md:p-8">
        <div className="max-w-6xl">
          <div className="mb-5 flex items-start justify-between gap-4 flex-wrap">
            <div className="min-w-0">
              {backHref && <BackLink href={backHref} label={backLabel} />}
              <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
              {subtitle && (
                <p className="text-sm text-gray-500 mt-0.5">{subtitle}</p>
              )}
            </div>
            {headerExtra}
          </div>
          <div className="mb-4">
            <QuickAdd
              projectId={quickAdd?.projectId}
              tagIds={quickAdd?.tagIds}
              flagged={quickAdd?.flagged}
              onCreated={append}
            />
          </div>
          <TaskList
            tasks={tasks}
            loading={loading}
            error={error}
            emptyMessage={emptyMessage}
            projects={projects}
            showProjectPicker={showProjectPicker}
            hideProject={hideProject}
            selectedId={selected?.id ?? null}
            onReorder={sortable ? reorder : undefined}
            onSelect={setSelected}
            onUpdated={handleUpdated}
            onCompleted={handleCompleted}
            onDeleted={handleDeleted}
          />
        </div>
      </div>
      <DockedPane
        open={!!selected}
        onClose={close}
        label="Task details"
        placeholder="Select a task to see its details."
      >
        {selected && (
          <TaskInspector
            task={selected}
            projects={projects}
            tags={tags}
            onCreateTag={addTag}
            onUpdated={handleUpdated}
            onCompleted={handleCompleted}
            onDeleted={handleDeleted}
            onClose={close}
          />
        )}
      </DockedPane>
    </div>
  );
}
