"use client";

import RequireAuth from "@/components/RequireAuth";
import GroupedTaskView from "@/components/GroupedTaskView";
import TagChip from "@/components/TagChip";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { groupByTag } from "@/lib/grouping";
import type { Tag, Task } from "@/lib/types";
import { useParams } from "next/navigation";
import { useCallback } from "react";
import Link from "next/link";

function TagContent({ tagId }: { tagId: string }) {
  const { tags } = useProjectsAndTags();
  const tag = tags.find((t) => t.id === tagId);
  const groupBy = useCallback(
    (tasks: Task[]) =>
      groupByTag(tasks, tags, (t: Tag) => <TagChip tag={t} size="md" />, tagId),
    [tags, tagId],
  );
  return (
    <GroupedTaskView
      title={tag ? tag.name : "Tag"}
      subtitle="Tasks carrying this tag."
      query={{ tag_id: tagId, view: "all" }}
      groupBy={groupBy}
      storageKey="focus:tags:showUnavailable"
      quickAdd={{ tagIds: [tagId] }}
      keep={(t) => t.status === "active" && t.tags.some((x) => x.id === tagId)}
      emptyMessage="No available tasks with this tag."
      headerExtra={
        <Link href="/tags" className="text-sm text-blue-600 hover:underline">
          All tags
        </Link>
      }
    />
  );
}

export default function TagTasksPage() {
  const params = useParams<{ id: string }>();
  return (
    <RequireAuth>
      <TagContent tagId={params.id} />
    </RequireAuth>
  );
}
