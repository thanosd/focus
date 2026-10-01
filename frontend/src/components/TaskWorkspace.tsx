"use client";

import type { Task } from "@/lib/types";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { useTasks, type TaskQuery } from "@/hooks/useTasks";
import { useCallback, useState } from "react";
import TaskList from "@/components/TaskList";
import TaskInspector from "@/components/TaskInspector";
import QuickAdd from "@/components/QuickAdd";

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
}

/**
 * A task list page body: heading, quick-add, list and inspector wired
 * together. Used by inbox, flagged, tag and project views.
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
}: TaskWorkspaceProps) {
  const { tasks, loading, error, upsert, remove, prepend } = useTasks(query);
  const { projects, tags, addTag } = useProjectsAndTags();
  const [selected, setSelected] = useState<Task | null>(null);

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
      if (next && (keep ? keep(next) : true)) prepend(next);
    },
    [keep, prepend, upsert, query.view],
  );

  const handleDeleted = useCallback(
    (id: string) => {
      remove(id);
      setSelected((s) => (s && s.id === id ? null : s));
    },
    [remove],
  );

  return (
    <div className="p-6 md:p-8 max-w-4xl">
      <div className="mb-5 flex items-start justify-between gap-4">
        <div>
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
          onCreated={(t) => {
            prepend(t);
            setSelected(t);
          }}
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
        onSelect={setSelected}
        onUpdated={handleUpdated}
        onCompleted={handleCompleted}
      />
      {selected && (
        <TaskInspector
          task={selected}
          projects={projects}
          tags={tags}
          onCreateTag={addTag}
          onUpdated={handleUpdated}
          onCompleted={handleCompleted}
          onDeleted={handleDeleted}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  );
}
