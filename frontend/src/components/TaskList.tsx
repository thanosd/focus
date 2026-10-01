"use client";

import type { Project, Task } from "@/lib/types";
import TaskRow from "@/components/TaskRow";

interface TaskListProps {
  tasks: Task[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  projects?: Project[];
  showProjectPicker?: boolean;
  hideProject?: boolean;
  selectedId?: string | null;
  onSelect: (task: Task) => void;
  onUpdated: (task: Task) => void;
  onCompleted?: (task: Task, next?: Task) => void;
}

export default function TaskList({
  tasks,
  loading,
  error,
  emptyMessage = "Nothing here.",
  projects,
  showProjectPicker,
  hideProject,
  selectedId,
  onSelect,
  onUpdated,
  onCompleted,
}: TaskListProps) {
  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
        {error}
      </div>
    );
  }
  if (loading) {
    return (
      <div className="bg-white border border-gray-200 rounded-lg divide-y divide-gray-100 animate-pulse">
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-11 px-3 py-3 flex items-center gap-3">
            <div className="w-5 h-5 rounded-full bg-gray-200" />
            <div className="h-3 bg-gray-200 rounded w-1/2" />
          </div>
        ))}
      </div>
    );
  }
  if (tasks.length === 0) {
    return (
      <div className="bg-white border border-dashed border-gray-200 rounded-lg p-8 text-center text-sm text-gray-500">
        {emptyMessage}
      </div>
    );
  }
  return (
    <div className="bg-white border border-gray-200 rounded-lg overflow-visible">
      {tasks.map((t) => (
        <TaskRow
          key={t.id}
          task={t}
          projects={projects}
          showProjectPicker={showProjectPicker}
          hideProject={hideProject}
          selected={selectedId === t.id}
          onSelect={onSelect}
          onUpdated={onUpdated}
          onCompleted={onCompleted}
        />
      ))}
    </div>
  );
}
