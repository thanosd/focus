"use client";

import RequireAuth from "@/components/RequireAuth";
import TaskWorkspace from "@/components/TaskWorkspace";
import { apiClient } from "@/lib/api-client";
import type { Tag } from "@/lib/types";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import Link from "next/link";

export default function TagTasksPage() {
  const params = useParams<{ id: string }>();
  const tagId = params.id;
  const [tag, setTag] = useState<Tag | null>(null);

  useEffect(() => {
    (async () => {
      const { data } = await apiClient.GET("/api/tags");
      setTag(data?.find((t) => t.id === tagId) ?? null);
    })();
  }, [tagId]);

  return (
    <RequireAuth>
      <TaskWorkspace
        title={tag ? tag.name : "Tag"}
        subtitle="Active tasks carrying this tag."
        query={{ tag_id: tagId }}
        quickAdd={{ tagIds: [tagId] }}
        emptyMessage="No active tasks with this tag."
        keep={(t) => t.status === "active" && t.tags.some((x) => x.id === tagId)}
        headerExtra={
          <Link href="/tags" className="text-sm text-blue-600 hover:underline">
            All tags
          </Link>
        }
      />
    </RequireAuth>
  );
}
