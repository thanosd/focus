"use client";

import RequireAuth from "@/components/RequireAuth";
import TaskWorkspace from "@/components/TaskWorkspace";

export default function FlaggedPage() {
  return (
    <RequireAuth>
      <TaskWorkspace
        title="Flagged"
        subtitle="Urgent tasks across every project."
        query={{ view: "flagged" }}
        quickAdd={{ flagged: true }}
        emptyMessage="Nothing is flagged."
        keep={(t) => t.flagged && t.status === "active"}
      />
    </RequireAuth>
  );
}
