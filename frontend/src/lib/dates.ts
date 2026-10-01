import { formatInTimeZone, fromZonedTime, toZonedTime } from "date-fns-tz";
import { differenceInCalendarDays, isBefore } from "date-fns";

/** Format an ISO timestamp in the user's timezone. */
export function fmt(iso: string, tz: string, pattern = "EEE, MMM d"): string {
  try {
    return formatInTimeZone(new Date(iso), tz, pattern);
  } catch {
    return iso;
  }
}

/** Short relative label like "Today", "Tomorrow", "Mon, Oct 5". */
export function dayLabel(iso: string, tz: string): string {
  const now = toZonedTime(new Date(), tz);
  const then = toZonedTime(new Date(iso), tz);
  const diff = differenceInCalendarDays(then, now);
  if (diff === 0) return "Today";
  if (diff === 1) return "Tomorrow";
  if (diff === -1) return "Yesterday";
  if (diff > 1 && diff < 7) return formatInTimeZone(new Date(iso), tz, "EEEE");
  const sameYear = then.getFullYear() === now.getFullYear();
  return formatInTimeZone(new Date(iso), tz, sameYear ? "MMM d" : "MMM d, yyyy");
}

/** Date + time label, omitting midnight. */
export function dateTimeLabel(iso: string, tz: string): string {
  const time = formatInTimeZone(new Date(iso), tz, "HH:mm");
  const day = dayLabel(iso, tz);
  return time === "00:00" ? day : `${day} ${formatInTimeZone(new Date(iso), tz, "h:mm a")}`;
}

export function isPast(iso: string): boolean {
  return isBefore(new Date(iso), new Date());
}

/** Value for an <input type="datetime-local"> in the user's timezone. */
export function toDatetimeLocal(iso: string | undefined, tz: string): string {
  if (!iso) return "";
  try {
    return formatInTimeZone(new Date(iso), tz, "yyyy-MM-dd'T'HH:mm");
  } catch {
    return "";
  }
}

/** Convert a datetime-local value (interpreted in tz) to an ISO string. */
export function fromDatetimeLocal(value: string, tz: string): string | null {
  if (!value) return null;
  try {
    return fromZonedTime(value, tz).toISOString();
  } catch {
    return null;
  }
}

/** Browser timezone, used before the user preference is loaded. */
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}
