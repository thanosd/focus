import type { Task } from "@/lib/types";
import { fmt } from "@/lib/dates";

/** "Nov 1, Dec 1, Jan 1" for a list of ISO timestamps in the user's zone. */
export function occurrenceList(
  dates: string[] | undefined,
  tz: string,
  max = 3,
): string {
  if (!dates || dates.length === 0) return "";
  const now = new Date();
  return dates
    .slice(0, max)
    .map((d) =>
      fmt(
        d,
        tz,
        new Date(d).getFullYear() === now.getFullYear()
          ? "MMM d"
          : "MMM d, yyyy",
      ),
    )
    .join(", ");
}

/** Tooltip text for a repeating task: what it repeats and when it's back. */
export function repeatSummary(task: Task, tz: string): string | null {
  if (!task.repeat_rule) return null;
  const desc = task.repeat_description ?? "on a schedule";
  const [first, second] = task.next_occurrences ?? [];
  let when = "";
  if (first) {
    when = ` · next ${fmt(first, tz, "MMM d")}`;
    if (second) when += `, then ${fmt(second, tz, "MMM d")}`;
  }
  return `Repeats ${desc}${when}`;
}

/** Toast line after completing a repeating task. */
export function comesBackLabel(next: Task, tz: string): string {
  const when = next.due_at ?? next.defer_until;
  return when
    ? `Completed — comes back ${fmt(when, tz, "EEE, MMM d")}`
    : "Completed — next occurrence created";
}
