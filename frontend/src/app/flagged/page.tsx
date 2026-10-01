"use client";

import RequireAuth from "@/components/RequireAuth";
import GroupedTaskView from "@/components/GroupedTaskView";
import TagChip from "@/components/TagChip";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { groupByTag } from "@/lib/grouping";
import type { Tag, Task } from "@/lib/types";
import { useCallback } from "react";
import Link from "next/link";

function FlaggedContent() {
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
      title="Flagged"
      subtitle="Urgent tasks you can act on right now, by context."
      query={{ view: "flagged" }}
      groupBy={groupBy}
      storageKey="focus:flagged:showUnavailable"
      quickAdd={{ flagged: true }}
      keep={(t) => t.flagged && t.status === "active"}
      emptyMessage="Nothing flagged is available."
    />
  );
}

export default function FlaggedPage() {
  return (
    <RequireAuth>
      <FlaggedContent />
    </RequireAuth>
  );
}
