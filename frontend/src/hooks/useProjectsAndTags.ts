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

/** Flattened project list (depth-first, any depth) with indentation info for <select> rendering. */
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
  const walk = (list: Project[], depth: number) => {
    for (const p of [...list].sort(sortFn)) {
      out.push({ project: p, depth });
      walk(byParent.get(p.id) ?? [], depth + 1);
    }
  };
  walk(top, 0);
  return out;
}

/** IDs of a project and everything nested under it. */
export function subtreeIds(projects: Project[], rootId: string): Set<string> {
  const ids = new Set<string>([rootId]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const p of projects) {
      if (p.parent_id && ids.has(p.parent_id) && !ids.has(p.id)) {
        ids.add(p.id);
        grew = true;
      }
    }
  }
  return ids;
}

/** Projects that may become a parent: depth 0 or 1, open, outside `excludeId`'s subtree. */
export function parentOptions(projects: Project[], excludeId?: string) {
  const excluded = excludeId ? subtreeIds(projects, excludeId) : new Set();
  return projectOptions(
    projects.filter((p) => p.status === "active" || p.status === "on_hold"),
  ).filter(({ project, depth }) => depth <= 1 && !excluded.has(project.id));
}
