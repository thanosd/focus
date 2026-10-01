"use client";

import RequireAuth from "@/components/RequireAuth";
import TaskWorkspace from "@/components/TaskWorkspace";

export default function InboxPage() {
  return (
    <RequireAuth>
      <TaskWorkspace
        title="Inbox"
        subtitle="Capture now, organize later. Assign a project to clear a task out of the inbox."
        query={{ view: "inbox" }}
        showProjectPicker
        sortable
        emptyMessage="Inbox zero. Nice."
        keep={(t) => !t.project_id && t.status === "active"}
      />
    </RequireAuth>
  );
}
