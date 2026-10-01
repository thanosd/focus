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
import { ListChecks, SkipForward } from "lucide-react";
import { Button } from "@/components/ui/button";

type Mode = "due" | "all";

function ReviewContent() {
  const { timezone } = useAuth();
  const { refreshCounts } = useCounts();
  const { toast } = useToast();
  const confirm = useConfirm();
  const { projects, tags, reload, addTag } = useProjectsAndTags();
  const [due, setDue] = useState<Project[]>([]);
  const [upcoming, setUpcoming] = useState<Project[]>([]);
  // A review session is a snapshot of projects to walk through, in order.
  const [queue, setQueue] = useState<Project[]>([]);
  const [index, setIndex] = useState(0);
  const [mode, setMode] = useState<Mode>("due");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const { data, error } = await apiClient.GET("/api/reviews");
    if (error || !data) {
      setError(errorMessage(error, "Failed to load reviews"));
      setLoading(false);
      return null;
    }
    setDue(data.due);
    setUpcoming(data.upcoming);
    setLoading(false);
    return data;
  }, []);

  useEffect(() => {
    (async () => {
      const data = await load();
      if (data) {
        setQueue(data.due);
        setIndex(0);
      }
    })();
  }, [load]);

  const startAll = () => {
    const all = [...due, ...upcoming];
    if (all.length === 0) return;
    setMode("all");
    setQueue(all);
    setIndex(0);
  };

  const finish = useCallback(() => {
    setMode("due");
    setQueue([]);
    setIndex(0);
    load();
  }, [load]);

  const current = queue[index];
  const total = queue.length;

  const advance = useCallback(async () => {
    setIndex((i) => i + 1);
    load();
    reload();
    refreshCounts();
  }, [load, reload, refreshCounts]);

  const onProjectChanged = useCallback(
    (p: Project) => {
      // A review stamp or status change means this one is done.
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

  const reviewAllButton = (
    <Button onClick={startAll} disabled={due.length + upcoming.length === 0}>
      <ListChecks className="w-4 h-4" /> Review all now
    </Button>
  );

  if (!current) {
    const finished = total;
    return (
      <div className="p-6 md:p-8 max-w-4xl space-y-4">
        <div className="flex items-end justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Review</h1>
            <p className="text-sm text-gray-500 mt-0.5">
              Walk through each project: is it still relevant, what is the next
              action, should it go on hold?
            </p>
          </div>
          {reviewAllButton}
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-8 text-center">
          <div className="text-3xl mb-2">✓</div>
          <div className="text-lg font-semibold text-gray-900">
            {due.length > 0 ? "Session complete" : "All caught up"}
          </div>
          <p className="text-sm text-gray-500 mt-1">
            {finished > 0
              ? `You went through ${finished} project${finished === 1 ? "" : "s"}.`
              : "No projects are due for review."}
          </p>
          {due.length > 0 && (
            <Button
              className="mt-4"
              variant="primary"
              onClick={() => {
                setMode("due");
                setQueue(due);
                setIndex(0);
              }}
            >
              Review {due.length} still due
            </Button>
          )}
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
      <div className="px-6 md:px-8 pt-6 md:pt-8 flex items-end justify-between gap-4 max-w-4xl flex-wrap">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            Review{mode === "all" ? " · all projects" : ""}
          </h1>
          <p className="text-sm text-gray-500 mt-0.5">
            Is it still relevant? What is the next action? Should it go on hold?
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-gray-500 tabular-nums">
            {Math.min(index + 1, total)} of {total}
          </span>
          <Button
            size="xs"
            variant="ghost"
            onClick={() => setIndex((i) => i + 1)}
          >
            <SkipForward className="w-3.5 h-3.5" /> Skip
          </Button>
          {mode === "all" && (
            <Button size="xs" variant="ghost" onClick={finish}>
              End session
            </Button>
          )}
          {mode === "due" && reviewAllButton}
        </div>
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
