"use client";

import { useMemo, useState } from "react";
import { Check, ChevronDown, Search } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface TimezonePickerProps {
  value: string;
  options: string[];
  onChange: (tz: string) => void;
  disabled?: boolean;
}

/** Searchable timezone combobox (hundreds of zones is too many for a Select). */
export default function TimezonePicker({
  value,
  options,
  onChange,
  disabled,
}: TimezonePickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase().replace(/ /g, "_");
    const list = q
      ? options.filter((z) => z.toLowerCase().includes(q))
      : options;
    return list.slice(0, 200);
  }, [options, query]);

  return (
    <Popover
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (!o) setQuery("");
      }}
    >
      <PopoverTrigger asChild>
        <button
          type="button"
          disabled={disabled}
          className={cn(
            "flex h-8 w-72 items-center justify-between gap-2 rounded-md border border-gray-300 bg-white px-2.5 text-sm text-gray-900 shadow-sm",
            "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50",
          )}
          aria-label="Timezone"
        >
          <span className="truncate">{value}</span>
          <ChevronDown className="h-4 w-4 shrink-0 text-gray-400" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-0">
        <div className="flex items-center gap-2 border-b border-gray-100 px-2.5 py-2">
          <Search className="h-3.5 w-3.5 text-gray-400" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search timezones…"
            autoFocus
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-gray-400"
          />
        </div>
        <div className="max-h-72 overflow-y-auto p-1">
          {filtered.map((z) => (
            <button
              key={z}
              type="button"
              onClick={() => {
                onChange(z);
                setOpen(false);
              }}
              className={cn(
                "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-left hover:bg-gray-100",
                z === value && "font-medium",
              )}
            >
              <span className="flex-1 truncate">{z}</span>
              {z === value && <Check className="h-3.5 w-3.5 text-blue-600" />}
            </button>
          ))}
          {filtered.length === 0 && (
            <div className="px-2 py-3 text-xs text-gray-400">No matches.</div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
