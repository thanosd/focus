"use client";

import type { RepeatRule } from "@/lib/types";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

interface RepeatEditorProps {
  value: RepeatRule | undefined;
  onChange: (rule: RepeatRule | null) => void;
  disabled?: boolean;
  /** Hide the enable checkbox (the parent already shows the rule exists). */
  hideToggle?: boolean;
}

const UNITS: RepeatRule["unit"][] = ["day", "week", "month", "year"];
const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const ANY_DAY = "__any__";

export function describeRepeat(rule: RepeatRule): string {
  if (rule.unit === "week" && rule.weekdays && rule.weekdays.length > 0) {
    const days = rule.weekdays.map((d) => WEEKDAYS[d]).join(", ");
    return rule.every === 1
      ? `every ${days}`
      : `every ${rule.every} weeks on ${days}`;
  }
  if (rule.unit === "month" && rule.day_of_month !== undefined) {
    const day =
      rule.day_of_month === -1 ? "the last day" : `day ${rule.day_of_month}`;
    return rule.every === 1
      ? `monthly on ${day}`
      : `every ${rule.every} months on ${day}`;
  }
  const unit = rule.every === 1 ? rule.unit : `${rule.every} ${rule.unit}s`;
  return `every ${unit}, from ${rule.from === "due" ? "due date" : "completion"}`;
}

/** Structured repeat rule editor: every N unit, anchors, and "from". */
export default function RepeatEditor({
  value,
  onChange,
  disabled,
  hideToggle = false,
}: RepeatEditorProps) {
  const enabled = !!value;
  const rule: RepeatRule = value ?? {
    every: 1,
    unit: "week",
    from: "completion",
  };

  const setUnit = (unit: RepeatRule["unit"]) => {
    // Anchors only make sense for their own unit.
    const next: RepeatRule = { ...rule, unit };
    if (unit !== "week") delete next.weekdays;
    if (unit !== "month") delete next.day_of_month;
    onChange(next);
  };

  const toggleWeekday = (d: number) => {
    const set = new Set(rule.weekdays ?? []);
    if (set.has(d)) set.delete(d);
    else set.add(d);
    const weekdays = Array.from(set).sort((a, b) => a - b);
    const next: RepeatRule = { ...rule };
    if (weekdays.length > 0) {
      next.weekdays = weekdays;
      next.from = "due";
    } else delete next.weekdays;
    onChange(next);
  };

  return (
    <div className="space-y-2">
      {!hideToggle && (
        <label className="flex items-center gap-2 text-sm text-gray-700">
          <Checkbox
            checked={enabled}
            disabled={disabled}
            onCheckedChange={(c) => onChange(c === true ? rule : null)}
          />
          Repeat
        </label>
      )}
      {(enabled || hideToggle) && (
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2 text-sm text-gray-700">
            <span>every</span>
            <Input
              type="number"
              min={1}
              value={rule.every}
              disabled={disabled}
              onChange={(e) =>
                onChange({
                  ...rule,
                  every: Math.max(1, Number(e.target.value) || 1),
                })
              }
              className="w-16"
            />
            <Select
              value={rule.unit}
              disabled={disabled}
              onValueChange={(v) => setUnit(v as RepeatRule["unit"])}
            >
              <SelectTrigger className="w-28" aria-label="Repeat unit">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {UNITS.map((u) => (
                  <SelectItem key={u} value={u}>
                    {rule.every === 1 ? u : `${u}s`}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {rule.unit === "week" && (
            <div
              className="flex flex-wrap gap-1"
              role="group"
              aria-label="Weekdays"
            >
              {WEEKDAYS.map((name, d) => {
                const on = rule.weekdays?.includes(d) ?? false;
                return (
                  <button
                    key={name}
                    type="button"
                    disabled={disabled}
                    aria-pressed={on}
                    onClick={() => toggleWeekday(d)}
                    className={cn(
                      "px-2 py-1 rounded-md border text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
                      on
                        ? "bg-blue-600 border-blue-600 text-white"
                        : "bg-white border-gray-300 text-gray-600 hover:bg-gray-50",
                    )}
                  >
                    {name}
                  </button>
                );
              })}
            </div>
          )}

          {rule.unit === "month" && (
            <div className="flex items-center gap-2 text-sm text-gray-700">
              <span>on</span>
              <Select
                value={
                  rule.day_of_month === undefined
                    ? ANY_DAY
                    : String(rule.day_of_month)
                }
                disabled={disabled}
                onValueChange={(v) => {
                  const next: RepeatRule = { ...rule };
                  if (v === ANY_DAY) delete next.day_of_month;
                  else {
                    next.day_of_month = Number(v);
                    next.from = "due";
                  }
                  onChange(next);
                }}
              >
                <SelectTrigger className="w-40" aria-label="Day of month">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ANY_DAY}>same day as last time</SelectItem>
                  {Array.from({ length: 31 }, (_, i) => i + 1).map((d) => (
                    <SelectItem key={d} value={String(d)}>
                      the {d}
                      {d % 10 === 1 && d !== 11
                        ? "st"
                        : d % 10 === 2 && d !== 12
                          ? "nd"
                          : d % 10 === 3 && d !== 13
                            ? "rd"
                            : "th"}
                    </SelectItem>
                  ))}
                  <SelectItem value="-1">the last day</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="flex items-center gap-2 text-sm text-gray-700">
            <span>from</span>
            <Select
              value={rule.from}
              disabled={disabled}
              onValueChange={(v) =>
                onChange({ ...rule, from: v as RepeatRule["from"] })
              }
            >
              <SelectTrigger className="w-40" aria-label="Repeat from">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="completion">completion date</SelectItem>
                <SelectItem value="due">due date</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      )}
    </div>
  );
}
