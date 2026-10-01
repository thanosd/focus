"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type {
  Project,
  RepeatRule,
  Tag,
  Task,
  UpdateTaskRequest,
} from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useCounts } from "@/contexts/CountsContext";
import { useToast } from "@/contexts/ToastContext";
import { dateTimeLabel, dayLabel } from "@/lib/dates";
import { useEffect, useRef, useState } from "react";
import { CloseIcon, FlagIcon } from "@/components/Icons";
import DeferMenu from "@/components/DeferMenu";
import DueEditor from "@/components/DueEditor";
import ProjectPicker from "@/components/ProjectPicker";
import TagPicker from "@/components/TagPicker";
import RepeatEditor, { describeRepeat } from "@/components/RepeatEditor";
import { HEADER_HEIGHT_PX } from "@/components/AppHeader";

interface TaskInspectorProps {
  task: Task;
  projects: Project[];
  tags: Tag[];
  onCreateTag: (name: string) => Promise<Tag | null>;
  onUpdated: (task: Task) => void;
  onDeleted: (taskId: string) => void;
  onCompleted?: (task: Task, next?: Task) => void;
  onClose: () => void;
}

/**
 * Right-side panel for editing every attribute of a task. Title and note
 * save on blur; everything else saves immediately.
 */
export default function TaskInspector({
  task,
  projects,
  tags,
  onCreateTag,
  onUpdated,
  onDeleted,
  onCompleted,
  onClose,
}: TaskInspectorProps) {
  const { timezone } = useAuth();
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const [title, setTitle] = useState(task.title);
  const [note, setNote] = useState(task.note);
  const [busy, setBusy] = useState(false);
  const titleRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setTitle(task.title);
    setNote(task.note);
  }, [task.id, task.title, task.note]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  const patch = async (body: UpdateTaskRequest): Promise<Task | null> => {
    setBusy(true);
    const { data, error } = await apiClient.PATCH("/api/tasks/{taskId}", {
      params: { path: { taskId: task.id } },
      body,
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, "Couldn't save task"), "error");
      return null;
    }
    onUpdated(data);
    refreshCounts();
    return data;
  };

  const action = async (kind: "complete" | "drop" | "reopen" | "delete") => {
    setBusy(true);
    if (kind === "delete") {
      const { error } = await apiClient.DELETE("/api/tasks/{taskId}", {
        params: { path: { taskId: task.id } },
      });
      setBusy(false);
      if (error) {
        toast(errorMessage(error, "Couldn't delete task"), "error");
        return;
      }
      onDeleted(task.id);
      refreshCounts();
      onClose();
      return;
    }
    if (kind === "complete") {
      const { data, error } = await apiClient.POST(
        "/api/tasks/{taskId}/complete",
        {
          params: { path: { taskId: task.id } },
        },
      );
      setBusy(false);
      if (error || !data) {
        toast(errorMessage(error, "Couldn't complete task"), "error");
        return;
      }
      if (data.next_task) toast("Next occurrence created", "success");
      if (onCompleted) onCompleted(data.task, data.next_task);
      else onUpdated(data.task);
      refreshCounts();
      return;
    }
    const path =
      kind === "drop"
        ? "/api/tasks/{taskId}/drop"
        : "/api/tasks/{taskId}/reopen";
    const { data, error } = await apiClient.POST(path, {
      params: { path: { taskId: task.id } },
    });
    setBusy(false);
    if (error || !data) {
      toast(errorMessage(error, `Couldn't ${kind} task`), "error");
      return;
    }
    onUpdated(data);
    refreshCounts();
  };

  const saveTitle = () => {
    const t = title.trim();
    if (!t) {
      setTitle(task.title);
      return;
    }
    if (t !== task.title) patch({ title: t });
  };

  const saveNote = () => {
    if (note !== task.note) patch({ note });
  };

  const isActive = task.status === "active";

  return (
    <aside
      className="fixed right-0 w-full sm:w-[26rem] bg-white border-l border-gray-200 shadow-xl z-40 flex flex-col"
      style={{
        top: `${HEADER_HEIGHT_PX}px`,
        height: `calc(100vh - ${HEADER_HEIGHT_PX}px)`,
      }}
      aria-label="Task details"
    >
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
        <div className="flex items-center gap-2">
          <span
            className={`text-[10px] uppercase tracking-wide font-semibold px-1.5 py-0.5 rounded ${
              task.status === "active"
                ? "bg-green-50 text-green-700"
                : task.status === "completed"
                  ? "bg-blue-50 text-blue-700"
                  : "bg-gray-100 text-gray-500"
            }`}
          >
            {task.status}
          </span>
          {!task.project_id && isActive && (
            <span className="text-[10px] uppercase tracking-wide font-semibold px-1.5 py-0.5 rounded bg-amber-50 text-amber-700">
              inbox
            </span>
          )}
        </div>
        <button
          onClick={onClose}
          className="p-1 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100"
          aria-label="Close"
        >
          <CloseIcon />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-5">
        <div className="flex items-start gap-2">
          <input
            ref={titleRef}
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onBlur={saveTitle}
            onKeyDown={(e) => {
              if (e.key === "Enter") (e.target as HTMLInputElement).blur();
            }}
            disabled={busy}
            className="flex-1 text-base font-medium text-gray-900 border-b border-transparent hover:border-gray-200 focus:border-blue-500 focus:outline-none py-1 bg-transparent"
          />
          <button
            type="button"
            onClick={() => patch({ flagged: !task.flagged })}
            disabled={busy}
            title={task.flagged ? "Flagged (urgent)" : "Flag as urgent"}
            className={`p-1.5 rounded-md ${
              task.flagged
                ? "text-red-500 bg-red-50"
                : "text-gray-300 hover:text-orange-500 hover:bg-gray-100"
            }`}
          >
            <FlagIcon className="w-5 h-5" filled={task.flagged} />
          </button>
        </div>

        <Field label="Project">
          <ProjectPicker
            projects={projects}
            value={task.project_id}
            disabled={busy}
            onChange={(id) => patch({ project_id: id })}
            className="w-full"
          />
        </Field>

        <Field label="Note">
          <textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            onBlur={saveNote}
            disabled={busy}
            rows={4}
            placeholder="Add a note…"
            className="w-full text-sm border border-gray-300 rounded-md px-2.5 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </Field>

        <Field label="Tags">
          <TagPicker
            tags={tags}
            selectedIds={task.tags.map((t) => t.id)}
            disabled={busy}
            onCreate={onCreateTag}
            onChange={(ids) => patch({ tag_ids: ids })}
          />
        </Field>

        <Field
          label="Defer"
          hint={
            task.defer_until
              ? `Deferred until ${dateTimeLabel(task.defer_until, timezone)}`
              : "Not deferred"
          }
        >
          <DeferMenu
            task={task}
            inline
            onUpdated={(t) => {
              onUpdated(t);
              refreshCounts();
            }}
          />
        </Field>

        <Field
          label="Due"
          hint={
            task.due_at
              ? `Due ${dateTimeLabel(task.due_at, timezone)}`
              : "No due date"
          }
        >
          <DueEditor task={task} onUpdated={onUpdated} />
        </Field>

        <Field
          label="Repeat"
          hint={task.repeat_rule ? describeRepeat(task.repeat_rule) : undefined}
        >
          <RepeatEditor
            value={task.repeat_rule}
            disabled={busy}
            onChange={(rule: RepeatRule | null) => patch({ repeat_rule: rule })}
          />
        </Field>

        <div className="text-xs text-gray-400 space-y-0.5 pt-2 border-t border-gray-100">
          <div>Created {dayLabel(task.created_at, timezone)}</div>
          {task.completed_at && (
            <div>Completed {dateTimeLabel(task.completed_at, timezone)}</div>
          )}
          {task.dropped_at && (
            <div>Dropped {dateTimeLabel(task.dropped_at, timezone)}</div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2 px-4 py-3 border-t border-gray-200 bg-gray-50">
        {isActive ? (
          <>
            <button
              onClick={() => action("complete")}
              disabled={busy}
              className="text-sm font-medium px-3 py-1.5 rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
            >
              Complete
            </button>
            <button
              onClick={() => action("drop")}
              disabled={busy}
              className="text-sm font-medium px-3 py-1.5 rounded-md border border-gray-300 bg-white text-gray-700 hover:bg-gray-100 disabled:opacity-50"
            >
              Drop
            </button>
          </>
        ) : (
          <button
            onClick={() => action("reopen")}
            disabled={busy}
            className="text-sm font-medium px-3 py-1.5 rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
          >
            Reopen
          </button>
        )}
        <button
          onClick={() => {
            if (window.confirm("Delete this task permanently?"))
              action("delete");
          }}
          disabled={busy}
          className="ml-auto text-sm font-medium px-3 py-1.5 rounded-md text-red-600 hover:bg-red-50 disabled:opacity-50"
        >
          Delete
        </button>
      </div>
    </aside>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="flex items-baseline justify-between mb-1.5">
        <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
          {label}
        </label>
        {hint && <span className="text-xs text-gray-500">{hint}</span>}
      </div>
      {children}
    </div>
  );
}
