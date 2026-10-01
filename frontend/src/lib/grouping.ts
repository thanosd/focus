import type { Tag, Task } from "@/lib/types";
import type { TaskGroup } from "@/components/GroupedTaskView";
import { differenceInCalendarDays } from "date-fns";
import { toZonedTime } from "date-fns-tz";

/** One section per tag (tag order), plus "No tag" at the end. */
export function groupByTag(
  tasks: Task[],
  tags: Tag[],
  render: (tag: Tag) => React.ReactNode,
  onlyTagId?: string,
): TaskGroup[] {
  const groups: TaskGroup[] = [];
  const list = onlyTagId ? tags.filter((t) => t.id === onlyTagId) : tags;
  for (const tag of list) {
    const inTag = tasks.filter((t) => t.tags.some((x) => x.id === tag.id));
    if (inTag.length > 0)
      groups.push({ key: tag.id, title: render(tag), tasks: inTag });
  }
  if (!onlyTagId) {
    const untagged = tasks.filter((t) => t.tags.length === 0);
    if (untagged.length > 0)
      groups.push({ key: "__none__", title: "No tag", tasks: untagged });
  }
  return groups;
}

/** Overdue / today / this week / later, by due date in the user's timezone. */
export function groupByDue(tasks: Task[], timezone: string): TaskGroup[] {
  const now = new Date();
  const zonedNow = toZonedTime(now, timezone);
  const buckets: Record<string, Task[]> = {
    overdue: [],
    today: [],
    week: [],
    later: [],
  };
  for (const t of tasks) {
    if (!t.due_at) continue;
    const due = new Date(t.due_at);
    if (due < now) {
      buckets.overdue.push(t);
      continue;
    }
    const days = differenceInCalendarDays(toZonedTime(due, timezone), zonedNow);
    if (days <= 0) buckets.today.push(t);
    else if (days < 7) buckets.week.push(t);
    else buckets.later.push(t);
  }
  const byDue = (a: Task, b: Task) =>
    new Date(a.due_at ?? 0).getTime() - new Date(b.due_at ?? 0).getTime();
  const out: TaskGroup[] = [];
  if (buckets.overdue.length)
    out.push({
      key: "overdue",
      title: "Overdue",
      tasks: buckets.overdue.sort(byDue),
      tone: "flex items-center gap-2 mb-2 text-xs font-semibold text-red-600 uppercase tracking-wide",
    });
  if (buckets.today.length)
    out.push({
      key: "today",
      title: "Due today",
      tasks: buckets.today.sort(byDue),
    });
  if (buckets.week.length)
    out.push({
      key: "week",
      title: "Due this week",
      tasks: buckets.week.sort(byDue),
    });
  if (buckets.later.length)
    out.push({
      key: "later",
      title: "Later",
      tasks: buckets.later.sort(byDue),
    });
  return out;
}
