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
  const rule: RepeatRule = value ?? {
    every: 1,
    unit: "week",
    from: "completion",
  };

  return (
    <div className="space-y-2">
      <label className="flex items-center gap-2 text-sm text-gray-700">
        <Checkbox
          checked={enabled}
          disabled={disabled}
          onCheckedChange={(c) => onChange(c === true ? rule : null)}
        />
        Repeat
      </label>
      {enabled && (
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
            onValueChange={(v) =>
              onChange({ ...rule, unit: v as RepeatRule["unit"] })
            }
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
      )}
    </div>
  );
}
