"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Task } from "@/lib/types";
import { useCallback, useEffect, useState } from "react";

export type TaskQuery = {
  view?: "inbox" | "available" | "flagged" | "due" | "completed" | "all";
  project_id?: string;
  tag_id?: string;
  q?: string;
};

/** Loads a task list for a query and exposes helpers to patch it locally. */
export function useTasks(query: TaskQuery) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const key = JSON.stringify(query);

  const reload = useCallback(async () => {
    const q = JSON.parse(key) as TaskQuery;
    const { data, error } = await apiClient.GET("/api/tasks", {
      params: { query: q },
    });
    if (error) {
      setError(errorMessage(error, "Failed to load tasks"));
    } else {
      setError(null);
      setTasks(data ?? []);
    }
    setLoading(false);
  }, [key]);

  useEffect(() => {
    setLoading(true);
    reload();
  }, [reload]);

  /** Replace a task in place (or drop it when it no longer belongs). */
  const upsert = useCallback((task: Task, keep = true) => {
    setTasks((list) => {
      const idx = list.findIndex((t) => t.id === task.id);
      if (!keep) return idx >= 0 ? list.filter((t) => t.id !== task.id) : list;
      if (idx < 0) return [...list, task];
      const next = [...list];
      next[idx] = task;
      return next;
    });
  }, []);

  const remove = useCallback((id: string) => {
    setTasks((list) => list.filter((t) => t.id !== id));
  }, []);

  const prepend = useCallback((task: Task) => {
    setTasks((list) => [task, ...list]);
  }, []);

  return { tasks, loading, error, reload, upsert, remove, prepend, setTasks };
}
