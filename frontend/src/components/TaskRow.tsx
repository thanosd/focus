"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Project, Task, UpdateTaskRequest } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useCounts } from "@/contexts/CountsContext";
import { useConfirm } from "@/contexts/ConfirmContext";
import { useToast } from "@/contexts/ToastContext";
import { dateTimeLabel, dayLabel, isPast } from "@/lib/dates";
import { unavailableReason } from "@/lib/availability";
import { useState } from "react";
import {
  Calendar,
  Check,
  Clock,
  Flag,
  Folder,
  GripVertical,
  MoreHorizontal,
  Repeat,
  RotateCcw,
  Trash2,
  XCircle,
} from "lucide-react";
import TagChip from "@/components/TagChip";
import DeferMenu from "@/components/DeferMenu";
import ProjectPicker from "@/components/ProjectPicker";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import Link from "next/link";

interface TaskRowProps {
  task: Task;
  projects?: Project[];
  /** Show the inline project dropdown (used in the inbox). */
  showProjectPicker?: boolean;
  /** Hide the project chip (e.g. inside a project's own list). */
  hideProject?: boolean;
  selected?: boolean;
  /** 1-based position shown for sequential projects. */
  index?: number;
  /** Props for the drag handle (from useSortable); renders the grip when set. */
  handleProps?: React.HTMLAttributes<HTMLButtonElement>;
  /** Visual state while being dragged. */
  dragging?: boolean;
  onSelect: (task: Task) => void;
  onUpdated: (task: Task) => void;
  onCompleted?: (task: Task, next?: Task) => void;
  onDeleted?: (taskId: string) => void;
}

export default function TaskRow({
  task,
  projects = [],
  showProjectPicker = false,
  hideProject = false,
  selected = false,
  index,
  handleProps,
  dragging = false,
  onSelect,
  onUpdated,
  onCompleted,
  onDeleted,
}: TaskRowProps) {
  const { timezone } = useAuth();
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const confirm = useConfirm();
  const [busy, setBusy] = useState(false);
  const [deferOpen, setDeferOpen] = useState(false);

  const done = task.status === "completed";
  const dropped = task.status === "dropped";
  const active = task.status === "active";
  const deferred = !!task.defer_until && !isPast(task.defer_until);
  const overdue = !!task.due_at && isPast(task.due_at) && active;
  const project = task.project_id
    ? projects.find((p) => p.id === task.project_id)
    : undefined;
  const reason = unavailableReason(task, project, timezone);
  const muted = active && !task.is_available;

  const complete = async () => {
    if (busy) return;
    setBusy(true);
    if (active) {
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
        { params: { path: { taskId: task.id } } },
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

  const drop = async () => {
    if (busy) return;
    setBusy(true);
    const { data, error } = await apiClient.POST("/api/tasks/{taskId}/drop", {
      params: { path: { taskId: task.id } },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't drop task"), "error");
      return;
    }
    onUpdated(data);
    refreshCounts();
  };

  const del = async () => {
    const ok = await confirm({
      title: "Delete this task?",
      description: `"${task.title}" will be permanently deleted.`,
      confirmLabel: "Delete",
      destructive: true,
    });
    if (!ok) return;
    setBusy(true);
    const { error } = await apiClient.DELETE("/api/tasks/{taskId}", {
      params: { path: { taskId: task.id } },
    });
    setBusy(false);
    if (error) {
      toast(errorMessage(error, "Couldn't delete task"), "error");
      return;
    }
    onDeleted?.(task.id);
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

  const stop = (e: React.SyntheticEvent) => e.stopPropagation();

  return (
    <div
      className={cn(
        "group relative flex items-start gap-2 px-3 py-2.5 border-b border-gray-100 last:border-b-0 cursor-pointer transition-colors border-l-4",
        selected ? "bg-blue-50" : "hover:bg-gray-50",
        task.flagged && active && !muted
          ? "border-l-red-400 bg-red-50/40"
          : "border-l-transparent",
        selected && task.flagged && active && "bg-blue-50",
        muted && !selected && "bg-gray-50/60",
        dragging && "shadow-lg ring-1 ring-blue-200 bg-white",
      )}
      onClick={() => onSelect(task)}
    >
      {handleProps && (
        <button
          type="button"
          {...handleProps}
          onClick={stop}
          aria-label="Drag to reorder"
          title="Drag to reorder"
          className={cn(
            "mt-0.5 -ml-1 p-0.5 rounded text-gray-300 hover:text-gray-500 cursor-grab active:cursor-grabbing touch-none flex-shrink-0",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
            "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
            dragging && "opacity-100 text-gray-500",
          )}
        >
          <GripVertical className="w-4 h-4" />
        </button>
      )}
      {index !== undefined && (
        <span
          className={cn(
            "mt-0.5 w-5 text-right text-xs tabular-nums flex-shrink-0 select-none",
            muted ? "text-gray-300" : "text-blue-600 font-semibold",
          )}
          aria-label={`Step ${index}`}
        >
          {index}
        </span>
      )}
      <button
        type="button"
        onClick={(e) => {
          stop(e);
          complete();
        }}
        disabled={busy}
        aria-label={done ? "Reopen task" : "Complete task"}
        className={cn(
          "mt-0.5 w-5 h-5 rounded-full border-2 flex items-center justify-center flex-shrink-0 transition-colors",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
          done
            ? "bg-blue-600 border-blue-600 text-white"
            : dropped
              ? "border-gray-300 bg-gray-100 text-transparent"
              : muted
                ? "border-gray-200 bg-gray-50 hover:border-blue-400 text-transparent hover:text-blue-200"
                : "border-gray-300 hover:border-blue-500 text-transparent hover:text-blue-300",
        )}
      >
        <Check className="w-3 h-3" strokeWidth={3} />
      </button>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "text-sm truncate",
              done || dropped
                ? "line-through text-gray-400"
                : muted
                  ? "text-gray-400"
                  : task.flagged
                    ? "font-medium text-gray-900"
                    : "text-gray-900",
            )}
          >
            {task.title}
          </span>
          {reason && (
            <span
              className="text-[10px] text-gray-400 italic whitespace-nowrap flex-shrink-0"
              title="Not available right now"
            >
              {reason}
            </span>
          )}
          {dropped && (
            <span className="text-[10px] uppercase tracking-wide text-gray-400 border border-gray-200 rounded px-1">
              dropped
            </span>
          )}
          {task.repeat_rule && (
            <Repeat className="w-3.5 h-3.5 text-gray-400 flex-shrink-0" />
          )}
        </div>
        {(task.tags.length > 0 ||
          task.defer_until ||
          task.due_at ||
          (!hideProject && task.project_id) ||
          showProjectPicker) && (
          <div
            className={cn(
              "mt-1 flex flex-wrap items-center gap-1.5",
              muted && "opacity-60",
            )}
          >
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
                  onClick={stop}
                  className="inline-flex items-center gap-1 text-[11px] text-gray-500 hover:text-blue-600"
                >
                  <Folder className="w-3 h-3" />
                  {task.project_name ?? "Project"}
                </Link>
              )
            )}
            {task.defer_until && (
              <span
                className={cn(
                  "inline-flex items-center gap-1 text-[11px] rounded px-1.5 py-0.5",
                  deferred
                    ? "bg-gray-100 text-gray-500"
                    : "bg-gray-50 text-gray-400",
                )}
                title={`Deferred until ${dateTimeLabel(task.defer_until, timezone)}`}
              >
                <Clock className="w-3 h-3" />
                {dayLabel(task.defer_until, timezone)}
              </span>
            )}
            {task.due_at && (
              <span
                className={cn(
                  "inline-flex items-center gap-1 text-[11px] rounded px-1.5 py-0.5",
                  overdue
                    ? "bg-red-100 text-red-700 font-medium"
                    : "bg-blue-50 text-blue-700",
                )}
                title={`Due ${dateTimeLabel(task.due_at, timezone)}`}
              >
                <Calendar className="w-3 h-3" />
                {dateTimeLabel(task.due_at, timezone)}
              </span>
            )}
            {task.tags.map((t) => (
              <TagChip key={t.id} tag={t} />
            ))}
          </div>
        )}
      </div>

      <div className="flex items-center gap-0.5 flex-shrink-0" onClick={stop}>
        <Popover open={deferOpen} onOpenChange={setDeferOpen}>
          <PopoverTrigger asChild>
            <button
              type="button"
              disabled={busy || !active}
              aria-label="Defer task"
              title="Defer"
              className={cn(
                "p-1 rounded-md transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
                deferOpen
                  ? "bg-gray-200 text-gray-700"
                  : "text-gray-300 hover:text-gray-600 hover:bg-gray-100 group-hover:text-gray-400",
                "disabled:opacity-30",
              )}
            >
              <Clock className="w-4 h-4" />
            </button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-80">
            <div className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
              Defer
            </div>
            <DeferMenu
              task={task}
              autoFocus
              onUpdated={(t) => {
                onUpdated(t);
                refreshCounts();
              }}
              onDone={() => setDeferOpen(false)}
            />
          </PopoverContent>
        </Popover>
        <button
          type="button"
          onClick={() => patch({ flagged: !task.flagged })}
          disabled={busy}
          aria-label={task.flagged ? "Remove flag" : "Flag as urgent"}
          title={task.flagged ? "Flagged (urgent)" : "Flag as urgent"}
          className={cn(
            "p-1 rounded-md transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
            task.flagged
              ? muted
                ? "text-red-300 hover:text-red-500"
                : "text-red-500 hover:text-red-600"
              : "text-gray-300 hover:text-orange-500 hover:bg-gray-100 group-hover:text-gray-400",
          )}
        >
          <Flag
            className="w-4 h-4"
            fill={task.flagged ? "currentColor" : "none"}
          />
        </button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              disabled={busy}
              aria-label="More actions"
              className="p-1 rounded-md text-gray-300 hover:text-gray-600 hover:bg-gray-100 group-hover:text-gray-400 data-[state=open]:bg-gray-200 data-[state=open]:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              <MoreHorizontal className="w-4 h-4" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            {active ? (
              <DropdownMenuItem onSelect={complete}>
                <Check /> Complete
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem onSelect={complete}>
                <RotateCcw /> Reopen
              </DropdownMenuItem>
            )}
            <DropdownMenuItem
              onSelect={() => patch({ flagged: !task.flagged })}
            >
              <Flag /> {task.flagged ? "Remove flag" : "Flag as urgent"}
            </DropdownMenuItem>
            {active && (
              <DropdownMenuItem onSelect={() => setDeferOpen(true)}>
                <Clock /> Defer…
              </DropdownMenuItem>
            )}
            {active && (
              <DropdownMenuItem onSelect={drop}>
                <XCircle /> Drop
              </DropdownMenuItem>
            )}
            {onDeleted && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem destructive onSelect={del}>
                  <Trash2 /> Delete
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
