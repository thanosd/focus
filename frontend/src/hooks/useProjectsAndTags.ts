"use client";

import { apiClient } from "@/lib/api-client";
import type { Project, Tag } from "@/lib/types";
import { useCallback, useEffect, useState } from "react";

/** Loads the project list and tag list used by pickers. */
export function useProjectsAndTags() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loaded, setLoaded] = useState(false);

  const reload = useCallback(async () => {
    const [p, t] = await Promise.all([
      apiClient.GET("/api/projects"),
      apiClient.GET("/api/tags"),
    ]);
    if (p.data) setProjects(p.data);
    if (t.data) setTags(t.data);
    setLoaded(true);
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  const addTag = useCallback(async (name: string): Promise<Tag | null> => {
    const { data } = await apiClient.POST("/api/tags", { body: { name } });
    if (data) {
      setTags((t) => [...t, data].sort((a, b) => a.name.localeCompare(b.name)));
      return data;
    }
    return null;
  }, []);

  return { projects, tags, loaded, reload, addTag };
}

/** Flattened project list with indentation info for <select> rendering. */
export function projectOptions(projects: Project[]) {
  const top = projects.filter((p) => !p.parent_id);
  const byParent = new Map<string, Project[]>();
  for (const p of projects) {
    if (p.parent_id) {
      const list = byParent.get(p.parent_id) ?? [];
      list.push(p);
      byParent.set(p.parent_id, list);
    }
  }
  const sortFn = (a: Project, b: Project) =>
    a.sort_order - b.sort_order || a.name.localeCompare(b.name);
  const out: { project: Project; depth: number }[] = [];
  for (const t of [...top].sort(sortFn)) {
    out.push({ project: t, depth: 0 });
    for (const c of (byParent.get(t.id) ?? []).sort(sortFn)) {
      out.push({ project: c, depth: 1 });
    }
  }
  return out;
}
