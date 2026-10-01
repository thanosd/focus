"use client";

import RequireAuth from "@/components/RequireAuth";
import ProjectView from "@/components/ProjectView";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { apiClient, errorMessage } from "@/lib/api-client";
import type { Project, ProjectStatus } from "@/lib/types";
import { useAuth } from "@/contexts/AuthContext";
import { useConfirm } from "@/contexts/ConfirmContext";
import { useCounts } from "@/contexts/CountsContext";
import { useToast } from "@/contexts/ToastContext";
import { dayLabel } from "@/lib/dates";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";

function ReviewContent() {
  const { timezone } = useAuth();
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const confirm = useConfirm();
  const { projects, tags, reload, addTag } = useProjectsAndTags();
  const [due, setDue] = useState<Project[]>([]);
  const [upcoming, setUpcoming] = useState<Project[]>([]);
  const [index, setIndex] = useState(0);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (resetProgress: boolean) => {
    const { data, error } = await apiClient.GET("/api/reviews");
    if (error || !data) {
      setError(errorMessage(error, "Failed to load reviews"));
    } else {
      setDue(data.due);
      setUpcoming(data.upcoming);
      if (resetProgress) {
        setIndex(0);
        setTotal(data.due.length);
      }
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    load(true);
  }, [load]);

  const current = due[0];

  // After the current project is reviewed / its status changes it leaves the
  // "due" list, so reloading naturally advances to the next one.
  const advance = useCallback(async () => {
    setIndex((i) => i + 1);
    await load(false);
    reload();
    refreshCounts();
  }, [load, reload, refreshCounts]);

  const onProjectChanged = useCallback(
    (p: Project) => {
      // A review stamp or status change means it is no longer due.
      if (
        p.status !== "active" ||
        (p.next_review_at && new Date(p.next_review_at) > new Date())
      ) {
        advance();
      }
    },
    [advance],
  );

  const setStatus = async (status: ProjectStatus) => {
    if (!current) return;
    if (status === "dropped") {
      const ok = await confirm({
        title: `Drop "${current.name}"?`,
        description: "The project and its tasks stay in the archive.",
        confirmLabel: "Drop project",
        destructive: true,
      });
      if (!ok) return;
    }
    const { error } = await apiClient.PATCH("/api/projects/{projectId}", {
      params: { path: { projectId: current.id } },
      body: { status },
    });
    if (error) {
      toast(errorMessage(error, "Couldn't update project"), "error");
      return;
    }
    toast(`Project ${status.replace("_", " ")}`);
    advance();
  };

  if (loading) return <div className="p-8 text-sm text-gray-500">Loading…</div>;
  if (error) {
    return (
      <div className="p-8">
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
        </div>
      </div>
    );
  }

  if (!current) {
    return (
      <div className="p-6 md:p-8 max-w-4xl space-y-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Review</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            Walk through each project: is it still relevant, what is the next
            action, should it go on hold?
          </p>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-8 text-center">
          <div className="text-3xl mb-2">✓</div>
          <div className="text-lg font-semibold text-gray-900">
            All caught up
          </div>
          <p className="text-sm text-gray-500 mt-1">
            {total > 0
              ? `You reviewed ${total} project${total === 1 ? "" : "s"}.`
              : "No projects are due for review."}
          </p>
        </div>
        {upcoming.length > 0 && (
          <div className="bg-white border border-gray-200 rounded-lg">
            <div className="px-4 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wide border-b border-gray-100">
              Upcoming reviews
            </div>
            {upcoming.map((p) => (
              <Link
                key={p.id}
                href={`/projects/${p.id}`}
                className="flex items-center gap-3 px-4 py-2.5 text-sm hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
              >
                <span className="flex-1 truncate text-gray-800">{p.name}</span>
                <span className="text-xs text-gray-500">
                  {p.next_review_at
                    ? dayLabel(p.next_review_at, timezone)
                    : "unscheduled"}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col">
      <div className="px-6 md:px-8 pt-6 md:pt-8 flex items-end justify-between max-w-4xl">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Review</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            Is it still relevant? What is the next action? Should it go on hold?
          </p>
        </div>
        {total > 0 && (
          <div className="text-sm text-gray-500 tabular-nums">
            {Math.min(index + 1, total)} of {total}
          </div>
        )}
      </div>
      <ProjectView
        key={current.id}
        projectId={current.id}
        projects={projects}
        tags={tags}
        onCreateTag={addTag}
        compact
        onProjectChanged={onProjectChanged}
        actions={
          <>
            <Button onClick={() => setStatus("on_hold")}>Put on hold</Button>
            <Button onClick={() => setStatus("completed")}>Complete</Button>
            <Button variant="danger-ghost" onClick={() => setStatus("dropped")}>
              Drop
            </Button>
          </>
        }
      />
    </div>
  );
}

export default function ReviewPage() {
  return (
    <RequireAuth>
      <ReviewContent />
    </RequireAuth>
  );
}
