"use client";

import RequireAuth from "@/components/RequireAuth";
import GroupedTaskView from "@/components/GroupedTaskView";
import { useAuth } from "@/contexts/AuthContext";
import { groupByDue } from "@/lib/grouping";
import type { Task } from "@/lib/types";
import { useCallback } from "react";

function DueContent() {
  const { timezone } = useAuth();
  const groupBy = useCallback(
    (tasks: Task[]) => groupByDue(tasks, timezone),
    [timezone],
  );
  return (
    <GroupedTaskView
      title="Due"
      subtitle="Everything with a due date, overdue first."
      query={{ view: "due" }}
      groupBy={groupBy}
      availabilityToggle={false}
      keep={(t) => t.status === "active" && !!t.due_at}
      emptyMessage="Nothing has a due date."
    />
  );
}

export default function DuePage() {
  return (
    <RequireAuth>
      <DueContent />
    </RequireAuth>
  );
}
