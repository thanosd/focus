"use client";

import type { Project, Task } from "@/lib/types";
import TaskRow from "@/components/TaskRow";
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { restrictToVerticalAxisIfAvailable } from "@/lib/dnd";

interface TaskListProps {
  tasks: Task[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  projects?: Project[];
  showProjectPicker?: boolean;
  hideProject?: boolean;
  /** Tag to omit from row chips (grouped tag views). */
  hideTagId?: string;
  /** Number active tasks 1, 2, 3… (sequential projects). */
  numbered?: boolean;
  /**
   * Enable drag-and-drop. Called with the full ordered list of active task
   * ids after a drop; return false to signal failure (the parent reverts).
   */
  onReorder?: (orderedActiveIds: string[]) => void;
  selectedId?: string | null;
  onSelect: (task: Task) => void;
  onUpdated: (task: Task) => void;
  onCompleted?: (task: Task, next?: Task) => void;
  onDeleted?: (taskId: string) => void;
}

type RowProps = Omit<
  TaskListProps,
  "tasks" | "loading" | "error" | "emptyMessage" | "numbered" | "onReorder"
> & { task: Task; index?: number };

function SortableRow({ task, index, selectedId, ...rest }: RowProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: task.id });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    position: "relative" as const,
    zIndex: isDragging ? 10 : undefined,
  };
  return (
    <div ref={setNodeRef} style={style}>
      <TaskRow
        task={task}
        index={index}
        selected={selectedId === task.id}
        handleProps={{ ...attributes, ...listeners }}
        dragging={isDragging}
        {...rest}
      />
    </div>
  );
}

export default function TaskList({
  tasks,
  loading,
  error,
  emptyMessage = "Nothing here.",
  projects,
  showProjectPicker,
  hideProject,
  hideTagId,
  numbered = false,
  onReorder,
  selectedId,
  onSelect,
  onUpdated,
  onCompleted,
  onDeleted,
}: TaskListProps) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

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

  const rowProps = {
    projects,
    showProjectPicker,
    hideProject,
    hideTagId,
    selectedId,
    onSelect,
    onUpdated,
    onCompleted,
    onDeleted,
  };

  // Active tasks come first and are the only sortable ones; completed or
  // dropped tasks (when shown) sit at the bottom.
  const activeTasks = tasks.filter((t) => t.status === "active");
  const inactiveTasks = tasks.filter((t) => t.status !== "active");
  let step = 0;
  const indexFor = (t: Task) => {
    if (!numbered || t.status !== "active") return undefined;
    step += 1;
    return step;
  };

  if (!onReorder) {
    return (
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        {[...activeTasks, ...inactiveTasks].map((t) => (
          <TaskRow
            key={t.id}
            task={t}
            index={indexFor(t)}
            selected={selectedId === t.id}
            {...rowProps}
          />
        ))}
      </div>
    );
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const ids = activeTasks.map((t) => t.id);
    const from = ids.indexOf(String(active.id));
    const to = ids.indexOf(String(over.id));
    if (from < 0 || to < 0) return;
    onReorder(arrayMove(ids, from, to));
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      modifiers={restrictToVerticalAxisIfAvailable}
      onDragEnd={handleDragEnd}
    >
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <SortableContext
          items={activeTasks.map((t) => t.id)}
          strategy={verticalListSortingStrategy}
        >
          {activeTasks.map((t) => (
            <SortableRow
              key={t.id}
              task={t}
              index={indexFor(t)}
              {...rowProps}
            />
          ))}
        </SortableContext>
        {inactiveTasks.map((t) => (
          <TaskRow
            key={t.id}
            task={t}
            selected={selectedId === t.id}
            {...rowProps}
          />
        ))}
      </div>
    </DndContext>
  );
}
