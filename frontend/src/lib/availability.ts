import type { Project, Task } from "@/lib/types";
import { dayLabel, isPast } from "@/lib/dates";

/**
 * Why an active task is not available right now. Mirrors the backend rule:
 * deferred into the future, project not active, or not first in a
 * sequential project.
 */
export function unavailableReason(
  task: Task,
  project: Project | undefined,
  timezone: string,
): string | null {
  if (task.status !== "active" || task.is_available) return null;
  if (task.defer_until && !isPast(task.defer_until)) {
    return `deferred until ${dayLabel(task.defer_until, timezone)}`;
  }
  if (project) {
    if (project.status === "on_hold") return "project on hold";
    if (project.status === "completed" || project.status === "dropped")
      return `project ${project.status}`;
    if (project.sequential) return "waiting on earlier task";
  }
  return "not available";
}

/** Reorder `list` so the tasks with the given ids come first, in that order. */
export function applyOrder<T extends { id: string }>(
  list: T[],
  orderedIds: string[],
): T[] {
  const byId = new Map(list.map((t) => [t.id, t]));
  const head = orderedIds.map((id) => byId.get(id)).filter((t): t is T => !!t);
  const seen = new Set(orderedIds);
  const tail = list.filter((t) => !seen.has(t.id));
  return [...head, ...tail];
}
