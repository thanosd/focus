"use client";

import { apiClient, errorMessage } from "@/lib/api-client";
import type { Tag } from "@/lib/types";
import { useConfirm } from "@/contexts/ConfirmContext";
import { useToast } from "@/contexts/ToastContext";
import { useEffect, useState } from "react";
import Link from "next/link";
import TagChip from "@/components/TagChip";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const PALETTE = [
  "#3b82f6",
  "#10b981",
  "#f59e0b",
  "#ef4444",
  "#8b5cf6",
  "#ec4899",
  "#14b8a6",
  "#6b7280",
];

/** Create / rename / recolor / delete tags (Settings → Tags). */
export default function TagManager() {
  const { toast } = useToast();
  const confirm = useConfirm();
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState("");
  const [color, setColor] = useState(PALETTE[0]);
  const [editing, setEditing] = useState<Tag | null>(null);
  const [editName, setEditName] = useState("");
  const [editColor, setEditColor] = useState("");

  useEffect(() => {
    (async () => {
      const { data, error } = await apiClient.GET("/api/tags");
      if (error) toast(errorMessage(error, "Failed to load tags"), "error");
      setTags(data ?? []);
      setLoading(false);
    })();
  }, [toast]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    const { data, error } = await apiClient.POST("/api/tags", {
      body: { name: name.trim(), color },
    });
    if (error || !data) {
      toast(errorMessage(error, "Couldn't create tag"), "error");
      return;
    }
    setTags((t) => [...t, data].sort((a, b) => a.name.localeCompare(b.name)));
    setName("");
    setColor(PALETTE[(PALETTE.indexOf(color) + 1) % PALETTE.length]);
  };

  const save = async () => {
    if (!editing) return;
    const { data, error } = await apiClient.PATCH("/api/tags/{tagId}", {
      params: { path: { tagId: editing.id } },
      body: { name: editName.trim() || editing.name, color: editColor },
    });
    if (error || !data) {
      toast(errorMessage(error, "Couldn't update tag"), "error");
      return;
    }
    setTags((t) => t.map((x) => (x.id === data.id ? data : x)));
    setEditing(null);
  };

  const del = async (tag: Tag) => {
    const ok = await confirm({
      title: `Delete tag "${tag.name}"?`,
      description: "Tasks keep their other tags.",
      confirmLabel: "Delete",
      destructive: true,
    });
    if (!ok) return;
    const { error } = await apiClient.DELETE("/api/tags/{tagId}", {
      params: { path: { tagId: tag.id } },
    });
    if (error) {
      toast(errorMessage(error, "Couldn't delete tag"), "error");
      return;
    }
    setTags((t) => t.filter((x) => x.id !== tag.id));
  };

  return (
    <div>
      <form
        onSubmit={create}
        className="flex items-center gap-2 bg-white border border-gray-300 rounded-md px-3 h-10 shadow-sm mb-4 focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-blue-500"
      >
        <input
          type="color"
          value={color}
          onChange={(e) => setColor(e.target.value)}
          className="w-6 h-6 rounded cursor-pointer border-0 bg-transparent p-0"
          aria-label="Tag color"
        />
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="New tag… (Enter to save)"
          className="flex-1 outline-none text-sm bg-transparent"
        />
        {name.trim() && (
          <Button type="submit" variant="primary" size="xs">
            Add
          </Button>
        )}
      </form>

      {loading ? (
        <div className="text-sm text-gray-500">Loading…</div>
      ) : tags.length === 0 ? (
        <div className="border border-dashed border-gray-200 rounded-lg p-6 text-center text-sm text-gray-500">
          No tags yet.
        </div>
      ) : (
        <div className="border border-gray-200 rounded-lg divide-y divide-gray-100">
          {tags.map((tag) => (
            <div key={tag.id} className="flex items-center gap-3 px-4 py-2.5">
              {editing?.id === tag.id ? (
                <>
                  <input
                    type="color"
                    value={editColor}
                    onChange={(e) => setEditColor(e.target.value)}
                    className="w-6 h-6 rounded cursor-pointer border-0 bg-transparent p-0"
                    aria-label="Tag color"
                  />
                  <Input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") save();
                      if (e.key === "Escape") setEditing(null);
                    }}
                    autoFocus
                    className="flex-1"
                  />
                  <Button size="xs" variant="primary" onClick={save}>
                    Save
                  </Button>
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => setEditing(null)}
                  >
                    Cancel
                  </Button>
                </>
              ) : (
                <>
                  <Link
                    href={`/tags/${tag.id}`}
                    className="flex items-center gap-3 flex-1 min-w-0"
                  >
                    <TagChip tag={tag} size="md" />
                    <span className="text-xs text-gray-500">
                      {tag.active_task_count ?? 0} active
                    </span>
                  </Link>
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => {
                      setEditing(tag);
                      setEditName(tag.name);
                      setEditColor(tag.color);
                    }}
                  >
                    Edit
                  </Button>
                  <Button
                    size="xs"
                    variant="danger-ghost"
                    onClick={() => del(tag)}
                  >
                    Delete
                  </Button>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
