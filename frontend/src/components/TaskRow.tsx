"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Project, Task, UpdateTaskRequest } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useCounts } from "@/contexts/CountsContext";
import { useToast } from "@/contexts/ToastContext";
import { dateTimeLabel, dayLabel, isPast } from "@/lib/dates";
import { useState } from "react";
import {
  CalendarIcon,
  CheckIcon,
  ClockIcon,
  FlagIcon,
  FolderIcon,
  RepeatIcon,
} from "@/components/Icons";
import TagChip from "@/components/TagChip";
import DeferMenu from "@/components/DeferMenu";
import ProjectPicker from "@/components/ProjectPicker";
import Link from "next/link";

interface TaskRowProps {
  task: Task;
  projects?: Project[];
  /** Show the inline project dropdown (used in the inbox). */
  showProjectPicker?: boolean;
  /** Hide the project chip (e.g. inside a project's own list). */
  hideProject?: boolean;
  selected?: boolean;
  onSelect: (task: Task) => void;
  onUpdated: (task: Task) => void;
  onCompleted?: (task: Task, next?: Task) => void;
}

export default function TaskRow({
  task,
  projects = [],
  showProjectPicker = false,
  hideProject = false,
  selected = false,
  onSelect,
  onUpdated,
  onCompleted,
}: TaskRowProps) {
  const { timezone } = useAuth();
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const [busy, setBusy] = useState(false);
  const [deferOpen, setDeferOpen] = useState(false);

  const done = task.status === "completed";
  const dropped = task.status === "dropped";
  const deferred = !!task.defer_until && !isPast(task.defer_until);
  const overdue =
    !!task.due_at && isPast(task.due_at) && task.status === "active";

  const complete = async () => {
    if (busy) return;
    setBusy(true);
    if (task.status === "active") {
      const { data, error } = await apiClient.POST(
        "/api/tasks/{taskId}/complete",
        { params: { path: { taskId: task.id } } },
      );
      setBusy(false);
      if (error || !data) {
        toast(errorMessage(error, "Couldn't complete task"), "error");
        return;
      }
      if (data.next_task) {
        toast(
          `Next occurrence created${
            data.next_task.defer_until
              ? ` for ${dayLabel(data.next_task.defer_until, timezone)}`
              : data.next_task.due_at
                ? ` due ${dayLabel(data.next_task.due_at, timezone)}`
                : ""
          }`,
          "success",
        );
      }
      onCompleted?.(data.task, data.next_task);
    } else {
      const { data, error } = await apiClient.POST(
        "/api/tasks/{taskId}/reopen",
        {
          params: { path: { taskId: task.id } },
        },
      );
      setBusy(false);
      if (error || !data) {
        toast(errorMessage(error, "Couldn't reopen task"), "error");
        return;
      }
      onUpdated(data);
    }
    refreshCounts();
  };

  const patch = async (body: UpdateTaskRequest) => {
    if (busy) return;
    setBusy(true);
    const { data, error } = await apiClient.PATCH("/api/tasks/{taskId}", {
      params: { path: { taskId: task.id } },
      body,
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't update task"), "error");
      return;
    }
    onUpdated(data);
    refreshCounts();
  };

  return (
    <div
      className={`group relative flex items-start gap-3 px-3 py-2.5 border-b border-gray-100 last:border-b-0 cursor-pointer transition-colors ${
        selected ? "bg-blue-50" : "hover:bg-gray-50"
      } ${task.flagged && !done && !dropped ? "border-l-4 border-l-red-400 bg-red-50/40" : "border-l-4 border-l-transparent"}`}
      onClick={() => onSelect(task)}
    >
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          complete();
        }}
        disabled={busy}
        aria-label={done ? "Reopen task" : "Complete task"}
        className={`mt-0.5 w-5 h-5 rounded-full border-2 flex items-center justify-center flex-shrink-0 transition-colors ${
          done
            ? "bg-blue-600 border-blue-600 text-white"
            : dropped
              ? "border-gray-300 bg-gray-100"
              : "border-gray-300 hover:border-blue-500 text-transparent hover:text-blue-300"
        }`}
      >
        <CheckIcon className="w-3 h-3" />
      </button>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span
            className={`text-sm truncate ${
              done || dropped
                ? "line-through text-gray-400"
                : task.flagged
                  ? "font-medium text-gray-900"
                  : "text-gray-900"
            }`}
          >
            {task.title}
          </span>
          {dropped && (
            <span className="text-[10px] uppercase tracking-wide text-gray-400 border border-gray-200 rounded px-1">
              dropped
            </span>
          )}
          {task.repeat_rule && (
            <RepeatIcon className="w-3.5 h-3.5 text-gray-400 flex-shrink-0" />
          )}
        </div>
        {(task.tags.length > 0 ||
          task.defer_until ||
          task.due_at ||
          (!hideProject && task.project_id) ||
          showProjectPicker) && (
          <div className="mt-1 flex flex-wrap items-center gap-1.5">
            {showProjectPicker ? (
              <ProjectPicker
                projects={projects}
                value={task.project_id}
                compact
                disabled={busy}
                onChange={(id) => patch({ project_id: id })}
              />
            ) : (
              !hideProject &&
              task.project_id && (
                <Link
                  href={`/projects/${task.project_id}`}
                  onClick={(e) => e.stopPropagation()}
                  className="inline-flex items-center gap-1 text-[11px] text-gray-500 hover:text-blue-600"
                >
                  <FolderIcon className="w-3 h-3" />
                  {task.project_name ?? "Project"}
                </Link>
              )
            )}
            {task.defer_until && (
              <span
                className={`inline-flex items-center gap-1 text-[11px] rounded px-1.5 py-0.5 ${
                  deferred
                    ? "bg-gray-100 text-gray-500"
                    : "bg-gray-50 text-gray-400"
                }`}
                title={`Deferred until ${dateTimeLabel(task.defer_until, timezone)}`}
              >
                <ClockIcon className="w-3 h-3" />
                {dayLabel(task.defer_until, timezone)}
              </span>
            )}
            {task.due_at && (
              <span
                className={`inline-flex items-center gap-1 text-[11px] rounded px-1.5 py-0.5 ${
                  overdue
                    ? "bg-red-100 text-red-700 font-medium"
                    : "bg-blue-50 text-blue-700"
                }`}
                title={`Due ${dateTimeLabel(task.due_at, timezone)}`}
              >
                <CalendarIcon className="w-3 h-3" />
                {dateTimeLabel(task.due_at, timezone)}
              </span>
            )}
            {task.tags.map((t) => (
              <TagChip key={t.id} tag={t} />
            ))}
          </div>
        )}
      </div>

      <div className="flex items-center gap-1 flex-shrink-0">
        <div className="relative">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              setDeferOpen((v) => !v);
            }}
            disabled={busy || done || dropped}
            aria-label="Defer task"
            title="Defer"
            className={`p-1 rounded-md transition-colors ${
              deferOpen
                ? "bg-gray-200 text-gray-700"
                : "text-gray-300 hover:text-gray-600 hover:bg-gray-100 group-hover:text-gray-400"
            } disabled:opacity-30`}
          >
            <ClockIcon className="w-4 h-4" />
          </button>
          {deferOpen && (
            <div
              onClick={(e) => e.stopPropagation()}
              className="absolute right-0"
            >
              <div className="relative -left-[19rem]">
                <DeferMenu
                  task={task}
                  onUpdated={(t) => {
                    onUpdated(t);
                    refreshCounts();
                  }}
                  onClose={() => setDeferOpen(false)}
                />
              </div>
            </div>
          )}
        </div>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            patch({ flagged: !task.flagged });
          }}
          disabled={busy}
          aria-label={task.flagged ? "Remove flag" : "Flag as urgent"}
          title={task.flagged ? "Flagged (urgent)" : "Flag as urgent"}
          className={`p-1 rounded-md transition-colors ${
            task.flagged
              ? "text-red-500 hover:text-red-600"
              : "text-gray-300 hover:text-orange-500 hover:bg-gray-100 group-hover:text-gray-400"
          }`}
        >
          <FlagIcon className="w-4 h-4" filled={task.flagged} />
        </button>
      </div>
    </div>
  );
}
