"use client";

import RequireAuth from "@/components/RequireAuth";
import { apiClient, errorMessage } from "@/lib/api-client";
import type { Tag } from "@/lib/types";
import { useToast } from "@/contexts/ToastContext";
import { useEffect, useState } from "react";
import Link from "next/link";
import TagChip from "@/components/TagChip";

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

function TagsContent() {
  const { toast } = useToast();
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState("");
  const [color, setColor] = useState(PALETTE[0]);
  const [editing, setEditing] = useState<Tag | null>(null);
  const [editName, setEditName] = useState("");
  const [editColor, setEditColor] = useState("");

  const load = async () => {
    const { data, error } = await apiClient.GET("/api/tags");
    if (error) toast(errorMessage(error, "Failed to load tags"), "error");
    setTags(data ?? []);
    setLoading(false);
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
    if (!window.confirm(`Delete tag "${tag.name}"? Tasks keep their other tags.`))
      return;
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
    <div className="p-6 md:p-8 max-w-3xl">
      <h1 className="text-2xl font-bold text-gray-900 mb-1">Tags</h1>
      <p className="text-sm text-gray-500 mb-5">
        Contexts, people, places — anything that cuts across projects.
      </p>

      <form
        onSubmit={create}
        className="flex items-center gap-2 bg-white border border-gray-200 rounded-lg px-3 py-2 mb-5 shadow-sm"
      >
        <input
          type="color"
          value={color}
          onChange={(e) => setColor(e.target.value)}
          className="w-7 h-7 rounded cursor-pointer border-0 bg-transparent"
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
          <button
            type="submit"
            className="text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md px-2.5 py-1"
          >
            Add
          </button>
        )}
      </form>

      {loading ? (
        <div className="text-sm text-gray-500">Loading…</div>
      ) : tags.length === 0 ? (
        <div className="bg-white border border-dashed border-gray-200 rounded-lg p-8 text-center text-sm text-gray-500">
          No tags yet.
        </div>
      ) : (
        <div className="bg-white border border-gray-200 rounded-lg divide-y divide-gray-100">
          {tags.map((tag) => (
            <div key={tag.id} className="flex items-center gap-3 px-4 py-3">
              {editing?.id === tag.id ? (
                <>
                  <input
                    type="color"
                    value={editColor}
                    onChange={(e) => setEditColor(e.target.value)}
                    className="w-7 h-7 rounded cursor-pointer border-0 bg-transparent"
                    aria-label="Tag color"
                  />
                  <input
                    type="text"
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") save();
                      if (e.key === "Escape") setEditing(null);
                    }}
                    autoFocus
                    className="flex-1 text-sm border border-gray-300 rounded-md px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <button
                    onClick={save}
                    className="text-xs font-medium px-2.5 py-1 rounded-md bg-blue-600 text-white hover:bg-blue-700"
                  >
                    Save
                  </button>
                  <button
                    onClick={() => setEditing(null)}
                    className="text-xs px-2.5 py-1 rounded-md text-gray-600 hover:bg-gray-100"
                  >
                    Cancel
                  </button>
                </>
              ) : (
                <>
                  <Link href={`/tags/${tag.id}`} className="flex items-center gap-3 flex-1 min-w-0">
                    <TagChip tag={tag} size="md" />
                    <span className="text-xs text-gray-500">
                      {tag.active_task_count ?? 0} active
                    </span>
                  </Link>
                  <button
                    onClick={() => {
                      setEditing(tag);
                      setEditName(tag.name);
                      setEditColor(tag.color);
                    }}
                    className="text-xs text-gray-500 hover:text-gray-800 px-2 py-1 rounded-md hover:bg-gray-100"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => del(tag)}
                    className="text-xs text-red-500 hover:text-red-700 px-2 py-1 rounded-md hover:bg-red-50"
                  >
                    Delete
                  </button>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function TagsPage() {
  return (
    <RequireAuth>
      <TagsContent />
    </RequireAuth>
  );
}
