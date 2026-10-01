"use client";

import type { Tag } from "@/lib/types";
import { useState } from "react";
import TagChip from "@/components/TagChip";

interface TagPickerProps {
  tags: Tag[];
  selectedIds: string[];
  onChange: (ids: string[]) => void;
  onCreate: (name: string) => Promise<Tag | null>;
  disabled?: boolean;
}

/** Chip multi-select with inline tag creation. */
export default function TagPicker({
  tags,
  selectedIds,
  onChange,
  onCreate,
  disabled,
}: TagPickerProps) {
  const [draft, setDraft] = useState("");
  const [creating, setCreating] = useState(false);
  const selected = tags.filter((t) => selectedIds.includes(t.id));
  const available = tags.filter((t) => !selectedIds.includes(t.id));

  const create = async () => {
    const name = draft.trim();
    if (!name) return;
    const existing = tags.find((t) => t.name.toLowerCase() === name.toLowerCase());
    if (existing) {
      if (!selectedIds.includes(existing.id)) onChange([...selectedIds, existing.id]);
      setDraft("");
      return;
    }
    setCreating(true);
    const tag = await onCreate(name);
    setCreating(false);
    if (tag) {
      onChange([...selectedIds, tag.id]);
      setDraft("");
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1.5 min-h-[1.5rem]">
        {selected.length === 0 && (
          <span className="text-xs text-gray-400">No tags</span>
        )}
        {selected.map((t) => (
          <TagChip
            key={t.id}
            tag={t}
            size="md"
            onRemove={
              disabled
                ? undefined
                : () => onChange(selectedIds.filter((id) => id !== t.id))
            }
          />
        ))}
      </div>
      {available.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {available.map((t) => (
            <button
              key={t.id}
              type="button"
              disabled={disabled}
              onClick={() => onChange([...selectedIds, t.id])}
              className="text-xs px-2 py-0.5 rounded-full border border-dashed border-gray-300 text-gray-600 hover:border-gray-400 hover:bg-gray-50 disabled:opacity-50"
            >
              + {t.name}
            </button>
          ))}
        </div>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          create();
        }}
        className="flex items-center gap-2"
      >
        <input
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="New tag…"
          disabled={disabled || creating}
          className="flex-1 text-sm border border-gray-300 rounded-md px-2.5 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          type="submit"
          disabled={disabled || creating || !draft.trim()}
          className="text-xs font-medium px-2.5 py-1 rounded-md border border-gray-300 bg-white hover:bg-gray-50 text-gray-700 disabled:opacity-50"
        >
          Add tag
        </button>
      </form>
    </div>
  );
}
