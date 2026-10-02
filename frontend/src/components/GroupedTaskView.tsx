"use client";

import type { Task } from "@/lib/types";
import { readBool, writeBool } from "@/lib/storage";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { useTasks, type TaskQuery } from "@/hooks/useTasks";
import { useCallback, useEffect, useMemo, useState } from "react";
import TaskList from "@/components/TaskList";
import TaskInspector from "@/components/TaskInspector";
import QuickAdd from "@/components/QuickAdd";
import DockedPane from "@/components/DockedPane";
import BackLink from "@/components/BackLink";
import { Switch } from "@/components/ui/switch";

export interface TaskGroup {
  key: string;
  title: React.ReactNode;
  tasks: Task[];
  /** Extra classes for the group heading (e.g. red for overdue). */
  tone?: string;
}

interface GroupedTaskViewProps {
  title: string;
  subtitle?: string;
  query: TaskQuery;
  /** Split the (already filtered) tasks into sections. */
  groupBy: (tasks: Task[]) => TaskGroup[];
  /**
   * Offer the "Show unavailable" toggle. Off = only tasks that can be acted
   * on now (not deferred, not blocked by sequence or an on-hold project).
   */
  availabilityToggle?: boolean;
  /** localStorage key for the toggle. */
  storageKey?: string;
  quickAdd?: { projectId?: string; tagIds?: string[]; flagged?: boolean };
  /** Decide whether an updated task still belongs in this view. */
  keep?: (task: Task) => boolean;
  emptyMessage?: string;
  headerExtra?: React.ReactNode;
  /** Phone-only back control in the header. */
  backHref?: string;
  backLabel?: string;
}

/**
 * Execution view: a list page whose tasks are rendered in sections (by tag,
 * by due date…), defaulting to what is actionable right now.
 */
export default function GroupedTaskView({
  title,
  subtitle,
  query,
  groupBy,
  availabilityToggle = true,
  storageKey = "focus:showUnavailable",
  quickAdd,
  keep,
  emptyMessage = "Nothing here.",
  headerExtra,
  backHref,
  backLabel = "Back",
}: GroupedTaskViewProps) {
  const { tasks, loading, error, upsert, remove, append } = useTasks(query);
  const { projects, tags, addTag } = useProjectsAndTags();
  const [selected, setSelected] = useState<Task | null>(null);
  const [showUnavailable, setShowUnavailable] = useState(false);

  useEffect(() => {
    setShowUnavailable(readBool(storageKey, false));
  }, [storageKey]);

  const toggle = (v: boolean) => {
    setShowUnavailable(v);
    writeBool(storageKey, v);
  };

  const visible = useMemo(() => {
    const active = tasks.filter((t) => t.status === "active");
    return availabilityToggle && !showUnavailable
      ? active.filter((t) => t.is_available)
      : active;
  }, [tasks, availabilityToggle, showUnavailable]);
  const hiddenCount =
    tasks.filter((t) => t.status === "active").length - visible.length;
  const groups = useMemo(() => groupBy(visible), [groupBy, visible]);

  const handleUpdated = useCallback(
    (task: Task) => {
      upsert(task, keep ? keep(task) : true);
      setSelected((s) => (s && s.id === task.id ? task : s));
    },
    [keep, upsert],
  );

  const handleCompleted = useCallback(
    (task: Task, next?: Task) => {
      upsert(task, false);
      setSelected((s) => (s && s.id === task.id ? task : s));
      if (next && (keep ? keep(next) : true)) append(next);
    },
    [keep, append, upsert],
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
            <div className="flex items-center gap-4 flex-wrap">
              {headerExtra}
              {availabilityToggle && (
                <label className="flex items-center gap-2 text-sm text-gray-600 select-none">
                  <Switch checked={showUnavailable} onCheckedChange={toggle} />
                  Show unavailable
                  {!showUnavailable && hiddenCount > 0 && (
                    <span className="text-xs text-gray-400">
                      ({hiddenCount})
                    </span>
                  )}
                </label>
              )}
            </div>
          </div>

          {quickAdd && (
            <div className="mb-4">
              <QuickAdd
                projectId={quickAdd.projectId}
                tagIds={quickAdd.tagIds}
                flagged={quickAdd.flagged}
                onCreated={append}
              />
            </div>
          )}

          {error ? (
            <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {error}
            </div>
          ) : loading ? (
            <TaskList
              tasks={[]}
              loading
              onSelect={setSelected}
              onUpdated={handleUpdated}
            />
          ) : groups.length === 0 ? (
            <div className="bg-white border border-dashed border-gray-200 rounded-lg p-8 text-center text-sm text-gray-500">
              {emptyMessage}
              {!showUnavailable && hiddenCount > 0 && (
                <div className="mt-1 text-xs text-gray-400">
                  {hiddenCount} unavailable task{hiddenCount === 1 ? "" : "s"}{" "}
                  hidden.
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-6">
              {groups.map((g) => (
                <section key={g.key}>
                  <div
                    className={
                      g.tone ??
                      "flex items-center gap-2 mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wide"
                    }
                  >
                    {g.title}
                    <span className="font-normal normal-case text-gray-400">
                      {g.tasks.length}
                    </span>
                  </div>
                  <TaskList
                    tasks={g.tasks}
                    projects={projects}
                    selectedId={selected?.id ?? null}
                    onSelect={setSelected}
                    onUpdated={handleUpdated}
                    onCompleted={handleCompleted}
                    onDeleted={handleDeleted}
                  />
                </section>
              ))}
            </div>
          )}
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
