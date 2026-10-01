"use client";

import type { RepeatRule } from "@/lib/types";

interface RepeatEditorProps {
  value: RepeatRule | undefined;
  onChange: (rule: RepeatRule | null) => void;
  disabled?: boolean;
}

const UNITS: RepeatRule["unit"][] = ["day", "week", "month", "year"];

export function describeRepeat(rule: RepeatRule): string {
  const unit = rule.every === 1 ? rule.unit : `${rule.every} ${rule.unit}s`;
  return `every ${unit}, from ${rule.from === "due" ? "due date" : "completion"}`;
}

/** Repeat rule editor: toggle + every N unit + anchor. */
export default function RepeatEditor({
  value,
  onChange,
  disabled,
}: RepeatEditorProps) {
  const enabled = !!value;
  const rule: RepeatRule = value ?? { every: 1, unit: "week", from: "completion" };

  return (
    <div className="space-y-2">
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          checked={enabled}
          disabled={disabled}
          onChange={(e) => onChange(e.target.checked ? rule : null)}
          className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
        />
        Repeat
      </label>
      {enabled && (
        <div className="flex flex-wrap items-center gap-2 text-sm text-gray-700">
          <span>every</span>
          <input
            type="number"
            min={1}
            value={rule.every}
            disabled={disabled}
            onChange={(e) =>
              onChange({ ...rule, every: Math.max(1, Number(e.target.value) || 1) })
            }
            className="w-16 border border-gray-300 rounded-md px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <select
            value={rule.unit}
            disabled={disabled}
            onChange={(e) =>
              onChange({ ...rule, unit: e.target.value as RepeatRule["unit"] })
            }
            className="border border-gray-300 rounded-md px-2 py-1 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {UNITS.map((u) => (
              <option key={u} value={u}>
                {rule.every === 1 ? u : `${u}s`}
              </option>
            ))}
          </select>
          <span>from</span>
          <select
            value={rule.from}
            disabled={disabled}
            onChange={(e) =>
              onChange({ ...rule, from: e.target.value as RepeatRule["from"] })
            }
            className="border border-gray-300 rounded-md px-2 py-1 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="completion">completion date</option>
            <option value="due">due date</option>
          </select>
        </div>
      )}
    </div>
  );
}
