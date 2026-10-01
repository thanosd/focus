"use client";

import RequireAuth from "@/components/RequireAuth";
import GroupedTaskView from "@/components/GroupedTaskView";
import TagChip from "@/components/TagChip";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { groupByTag } from "@/lib/grouping";
import type { Tag, Task } from "@/lib/types";
import { useCallback } from "react";
import Link from "next/link";

function TagsContent() {
  const { tags } = useProjectsAndTags();
  const groupBy = useCallback(
    (tasks: Task[]) =>
      groupByTag(tasks, tags, (tag: Tag) => (
        <Link href={`/tags/${tag.id}`} className="hover:opacity-80">
          <TagChip tag={tag} size="md" />
        </Link>
      )),
    [tags],
  );
  return (
    <GroupedTaskView
      title="Tags"
      subtitle="What you can act on right now, by context."
      query={{ view: "all" }}
      groupBy={groupBy}
      storageKey="focus:tags:showUnavailable"
      keep={(t) => t.status === "active"}
      emptyMessage="No available tasks."
      headerExtra={
        <Link
          href="/settings#tags"
          className="text-sm text-blue-600 hover:underline"
        >
          Manage tags
        </Link>
      }
    />
  );
}

export default function TagsPage() {
  return (
    <RequireAuth>
      <TagsContent />
    </RequireAuth>
  );
}
