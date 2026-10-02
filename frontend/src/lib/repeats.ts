import type { Task } from "@/lib/types";
import { fmt } from "@/lib/dates";

// Occurrences always carry the year: a repeat that comes back "Jan 1"
// is ambiguous the moment the list crosses a year boundary.
const OCCURRENCE_FORMAT = "MMM d, yyyy";

/** "Nov 1, 2026, Dec 1, 2026, Jan 1, 2027" for ISO timestamps in the user's zone. */
export function occurrenceList(
  dates: string[] | undefined,
  tz: string,
  max = 3,
): string {
  if (!dates || dates.length === 0) return "";
  return dates
    .slice(0, max)
    .map((d) => fmt(d, tz, OCCURRENCE_FORMAT))
    .join(" · ");
}

/** Tooltip text for a repeating task: what it repeats and when it's back. */
export function repeatSummary(task: Task, tz: string): string | null {
  if (!task.repeat_rule) return null;
  const desc = task.repeat_description ?? "on a schedule";
  const [first, second] = task.next_occurrences ?? [];
  let when = "";
  if (first) {
    when = ` · next ${fmt(first, tz, OCCURRENCE_FORMAT)}`;
    if (second) when += `, then ${fmt(second, tz, OCCURRENCE_FORMAT)}`;
  }
  return `Repeats ${desc}${when}`;
}

/** Toast line after completing a repeating task. */
export function comesBackLabel(next: Task, tz: string): string {
  const when = next.due_at ?? next.defer_until;
  return when
    ? `Completed — comes back ${fmt(when, tz, "EEE, MMM d, yyyy")}`
    : "Completed — next occurrence created";
}
