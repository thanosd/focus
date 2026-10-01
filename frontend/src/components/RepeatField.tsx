"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { components } from "@/lib/api-types";
import type { RepeatRule, Task } from "@/lib/types";
import { occurrenceList } from "@/lib/repeats";
import { useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, Repeat } from "lucide-react";
import RepeatEditor from "@/components/RepeatEditor";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

type ParseRepeatResponse = components["schemas"]["ParseRepeatResponse"];

interface RepeatFieldProps {
  task: Task;
  timezone: string;
  disabled?: boolean;
  onUpdated: (task: Task) => void;
}

/**
 * Repeat control: type a schedule in plain English ("first of every
 * month", "every other friday"), see what it means before applying, and
 * tweak the details in the structured editor if needed.
 */
export default function RepeatField({
  task,
  timezone,
  disabled = false,
  onUpdated,
}: RepeatFieldProps) {
  const [input, setInput] = useState("");
  const [preview, setPreview] = useState<ParseRepeatResponse | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [applying, setApplying] = useState(false);
  const [applyError, setApplyError] = useState<string | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const seq = useRef(0);

  useEffect(() => {
    setInput("");
    setPreview(null);
    setPreviewError(null);
    setApplyError(null);
    setShowDetails(false);
  }, [task.id]);

  // Debounced preview while typing.
  useEffect(() => {
    const phrase = input.trim();
    if (!phrase) {
      setPreview(null);
      setPreviewError(null);
      setPreviewing(false);
      return;
    }
    const id = ++seq.current;
    setPreviewing(true);
    const timer = setTimeout(async () => {
      const { data, error } = await apiClient.POST("/api/repeats/parse", {
        body: { input: phrase, timezone },
      });
      if (id !== seq.current) return;
      setPreviewing(false);
      if (error || !data) {
        setPreview(null);
        setPreviewError(errorMessage(error, "Couldn't understand that"));
        return;
      }
      setPreviewError(null);
      setPreview(data);
    }, 400);
    return () => clearTimeout(timer);
  }, [input, timezone]);

  const send = async (body: components["schemas"]["SetRepeatRequest"]) => {
    setApplying(true);
    setApplyError(null);
    const { data, error } = await apiClient.POST("/api/tasks/{taskId}/repeat", {
      params: { path: { taskId: task.id } },
      body: { ...body, timezone },
    });
    setApplying(false);
    if (error || !data) {
      setApplyError(errorMessage(error, "Couldn't set repeat"));
      return;
    }
    onUpdated(data);
    setInput("");
    setPreview(null);
    setPreviewError(null);
  };

  const apply = () => {
    const phrase = input.trim();
    if (!phrase || applying) return;
    send({ input: phrase });
  };

  const busy = disabled || applying;
  const next = occurrenceList(task.next_occurrences, timezone);

  return (
    <div className="space-y-2">
      <Input
        value={input}
        disabled={busy}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            apply();
          }
        }}
        placeholder={
          task.repeat_rule
            ? "Change… e.g. every other friday"
            : "Repeat… e.g. first of every month, every other friday"
        }
        aria-label="Repeat schedule"
      />
      {input.trim() && (
        <div
          className={cn(
            "text-xs rounded-md px-2 py-1.5",
            previewError
              ? "bg-red-50 text-red-700"
              : "bg-blue-50 text-blue-800",
          )}
        >
          {previewError ? (
            previewError
          ) : preview ? (
            <>
              <span className="font-medium">{preview.description}</span>
              {preview.next_occurrences.length > 0 && (
                <span className="text-blue-700">
                  {" "}
                  · {occurrenceList(preview.next_occurrences, timezone)}
                </span>
              )}
              <span className="text-blue-600/70"> — press Enter to apply</span>
            </>
          ) : previewing ? (
            "Working it out…"
          ) : null}
        </div>
      )}
      {applyError && (
        <div className="text-xs text-red-700 bg-red-50 rounded-md px-2 py-1.5">
          {applyError}
        </div>
      )}

      {task.repeat_rule ? (
        <div className="rounded-md border border-gray-200 bg-gray-50 px-3 py-2 space-y-1">
          <div className="flex items-center gap-2 text-sm text-gray-900">
            <Repeat className="w-4 h-4 text-blue-600 flex-shrink-0" />
            <span className="font-medium capitalize">
              {task.repeat_description ?? "Repeats"}
            </span>
          </div>
          {next && (
            <div className="text-xs text-gray-600 pl-6">Next: {next}</div>
          )}
          <div className="flex items-center gap-1 pl-5 pt-1">
            <Button
              size="xs"
              variant="ghost"
              disabled={busy}
              onClick={() => setShowDetails((v) => !v)}
              aria-expanded={showDetails}
            >
              {showDetails ? (
                <ChevronDown className="w-3.5 h-3.5" />
              ) : (
                <ChevronRight className="w-3.5 h-3.5" />
              )}
              Edit details
            </Button>
            <Button
              size="xs"
              variant="danger-ghost"
              disabled={busy}
              onClick={() => send({ clear: true })}
            >
              Remove repeat
            </Button>
          </div>
          {showDetails && (
            <div className="pt-2 pl-5">
              <RepeatEditor
                hideToggle
                value={task.repeat_rule}
                disabled={busy}
                onChange={(rule: RepeatRule | null) =>
                  rule ? send({ rule }) : send({ clear: true })
                }
              />
            </div>
          )}
        </div>
      ) : (
        <p className="text-xs text-gray-400">
          Doesn&apos;t repeat. Try &ldquo;weekdays&rdquo;, &ldquo;every 2 weeks
          after completion&rdquo; or &ldquo;on the 15th of each month&rdquo;.
        </p>
      )}
    </div>
  );
}
