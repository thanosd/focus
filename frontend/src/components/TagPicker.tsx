"use client";

import type { Tag } from "@/lib/types";
import { useMemo, useState } from "react";
import { Check, Plus, Search } from "lucide-react";
import TagChip from "@/components/TagChip";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface TagPickerProps {
  tags: Tag[];
  selectedIds: string[];
  onChange: (ids: string[]) => void;
  onCreate: (name: string) => Promise<Tag | null>;
  disabled?: boolean;
}

/** Chip list plus a popover multi-select with search and inline creation. */
export default function TagPicker({
  tags,
  selectedIds,
  onChange,
  onCreate,
  disabled,
}: TagPickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [creating, setCreating] = useState(false);
  const selected = tags.filter((t) => selectedIds.includes(t.id));

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return q ? tags.filter((t) => t.name.toLowerCase().includes(q)) : tags;
  }, [tags, query]);
  const exact = tags.some(
    (t) => t.name.toLowerCase() === query.trim().toLowerCase(),
  );

  const toggle = (id: string) => {
    onChange(
      selectedIds.includes(id)
        ? selectedIds.filter((x) => x !== id)
        : [...selectedIds, id],
    );
  };

  const create = async () => {
    const name = query.trim();
    if (!name || exact) return;
    setCreating(true);
    const tag = await onCreate(name);
    setCreating(false);
    if (tag) {
      onChange([...selectedIds, tag.id]);
      setQuery("");
    }
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5 min-h-[2rem]">
      {selected.map((t) => (
        <TagChip
          key={t.id}
          tag={t}
          size="md"
          onRemove={disabled ? undefined : () => toggle(t.id)}
        />
      ))}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button size="xs" variant="ghost" disabled={disabled}>
            <Plus className="h-3.5 w-3.5" />
            {selected.length === 0 ? "Add tags" : "Add"}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-64 p-0">
          <div className="flex items-center gap-2 border-b border-gray-100 px-2.5 py-2">
            <Search className="h-3.5 w-3.5 text-gray-400" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  if (filtered.length === 1 && !exact && !query.trim()) return;
                  if (exact) toggle(filtered[0]?.id ?? "");
                  else if (query.trim()) create();
                }
              }}
              placeholder="Search or create…"
              autoFocus
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-gray-400"
            />
          </div>
          <div className="max-h-56 overflow-y-auto p-1">
            {filtered.map((t) => {
              const on = selectedIds.includes(t.id);
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => toggle(t.id)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-left hover:bg-gray-100",
                    on && "font-medium",
                  )}
                >
                  <span
                    className="h-2.5 w-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: t.color }}
                  />
                  <span className="flex-1 truncate">{t.name}</span>
                  {on && <Check className="h-3.5 w-3.5 text-blue-600" />}
                </button>
              );
            })}
            {filtered.length === 0 && !query.trim() && (
              <div className="px-2 py-3 text-xs text-gray-400">
                No tags yet.
              </div>
            )}
            {query.trim() && !exact && (
              <button
                type="button"
                onClick={create}
                disabled={creating}
                className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-left text-blue-600 hover:bg-blue-50 disabled:opacity-50"
              >
                <Plus className="h-3.5 w-3.5" />
                Create &ldquo;{query.trim()}&rdquo;
              </button>
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
