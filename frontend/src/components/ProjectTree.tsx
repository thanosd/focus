"use client";

import type { Project, ProjectStatus } from "@/lib/types";
import Link from "next/link";
import { useState } from "react";
import {
  CheckCircle2,
  ChevronRight,
  GripVertical,
  MoreHorizontal,
  PauseCircle,
  PlayCircle,
  Trash2,
  XCircle,
} from "lucide-react";
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
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { restrictToVerticalAxisIfAvailable } from "@/lib/dnd";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

interface ProjectTreeProps {
  projects: Project[];
  selectedId?: string | null;
  /** Show on_hold / completed / dropped projects too. */
  showInactive: boolean;
  onToggleInactive: (v: boolean) => void;
  /** Row menu actions; the menu is hidden when these are absent. */
  onSetStatus?: (project: Project, status: ProjectStatus) => void;
  onDelete?: (project: Project) => void;
  /**
   * Drag-and-drop among siblings. `moved` was dropped on `over` (same
   * parent); `after` says whether it lands below `over`. The drag handle
   * is hidden when this is absent.
   */
  onReorder?: (moved: Project, over: Project, after: boolean) => void;
}

/** One tree row (plus its subtree), sortable within its sibling group. */
function TreeRow({
  project: p,
  depth,
  children: subtree,
  canDrag,
  render,
}: {
  project: Project;
  depth: number;
  children: React.ReactNode;
  canDrag: boolean;
  render: (
    p: Project,
    depth: number,
    handle: React.ReactNode,
    dragging: boolean,
  ) => React.ReactNode;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: p.id, disabled: !canDrag });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    position: "relative" as const,
    zIndex: isDragging ? 10 : undefined,
  };
  const handle = canDrag ? (
    <button
      type="button"
      {...attributes}
      {...listeners}
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      aria-label={`Drag to reorder ${p.name}`}
      title="Drag to reorder"
      className={cn(
        "-ml-1 p-0.5 rounded text-gray-300 hover:text-gray-500 cursor-grab active:cursor-grabbing touch-none flex-shrink-0",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500",
        "opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100",
        isDragging && "opacity-100 text-gray-500",
      )}
    >
      <GripVertical className="w-3.5 h-3.5" />
    </button>
  ) : null;
  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "group/row relative rounded-md",
        isDragging && "shadow-lg ring-1 ring-blue-200 bg-white",
      )}
    >
      {render(p, depth, handle, isDragging)}
      {subtree}
    </div>
  );
}

export function StatusBadge({ status }: { status: Project["status"] }) {
  if (status === "active") return null;
  const cls = {
    on_hold: "bg-amber-50 text-amber-700",
    completed: "bg-blue-50 text-blue-700",
    dropped: "bg-gray-100 text-gray-500",
  }[status];
  return (
    <span
      className={cn(
        "text-[10px] uppercase tracking-wide font-semibold px-1.5 py-0.5 rounded",
        cls,
      )}
    >
      {status.replace("_", " ")}
    </span>
  );
}

/** Three-level tree of projects (bucket > project > sub-project) with task counts. */
export default function ProjectTree({
  projects,
  selectedId,
  showInactive,
  onToggleInactive,
  onSetStatus,
  onDelete,
  onReorder,
}: ProjectTreeProps) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const visible = showInactive
    ? projects
    : projects.filter((p) => p.status === "active" || p.id === selectedId);
  const sortFn = (a: Project, b: Project) =>
    a.sort_order - b.sort_order || a.name.localeCompare(b.name);
  const top = visible.filter((p) => !p.parent_id).sort(sortFn);
  const childrenOf = (id: string) =>
    visible.filter((p) => p.parent_id === id).sort(sortFn);
  // Children whose parent is hidden by the filter still need a home.
  const orphans = visible
    .filter((p) => p.parent_id && !visible.some((x) => x.id === p.parent_id))
    .sort(sortFn);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!onReorder || !over || active.id === over.id) return;
    const moved = visible.find((p) => p.id === active.id);
    const target = visible.find((p) => p.id === over.id);
    if (!moved || !target) return;
    if ((moved.parent_id ?? null) !== (target.parent_id ?? null)) return;
    const siblings = moved.parent_id ? childrenOf(moved.parent_id) : top;
    const from = siblings.findIndex((p) => p.id === moved.id);
    const to = siblings.findIndex((p) => p.id === target.id);
    if (from < 0 || to < 0) return;
    onReorder(moved, target, from < to);
  };

  const renderRow = (
    p: Project,
    depth: number,
    handle: React.ReactNode,
    dragging: boolean,
  ) => {
    const active = p.id === selectedId;
    const kids = childrenOf(p.id);
    const isCollapsed = collapsed.has(p.id);
    const closed = p.status === "completed" || p.status === "dropped";
    return (
      <>
        <Link
          href={`/projects/${p.id}`}
          onClick={(e) => {
            if (dragging) e.preventDefault();
          }}
          className={cn(
            "flex items-center gap-2 py-1.5 pr-8 rounded-md text-sm transition-colors",
            active
              ? "bg-blue-50 text-blue-700 font-medium"
              : "text-gray-700 hover:bg-gray-100",
          )}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
        >
          {handle}
          {kids.length > 0 ? (
            <button
              type="button"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setCollapsed((s) => {
                  const n = new Set(s);
                  if (n.has(p.id)) n.delete(p.id);
                  else n.add(p.id);
                  return n;
                });
              }}
              className="w-4 h-4 flex items-center justify-center text-gray-400 rounded hover:text-gray-600"
              aria-label={isCollapsed ? "Expand" : "Collapse"}
            >
              <ChevronRight
                className={cn(
                  "w-3.5 h-3.5 transition-transform",
                  !isCollapsed && "rotate-90",
                )}
              />
            </button>
          ) : (
            <span className="w-4 h-4" />
          )}
          <span className="truncate flex-1">{p.name}</span>
          <StatusBadge status={p.status} />
          <span
            className="text-[11px] text-gray-400 tabular-nums"
            title={`${p.available_task_count} available / ${p.remaining_task_count} remaining`}
          >
            {p.available_task_count}/{p.remaining_task_count}
          </span>
        </Link>
        {(onSetStatus || onDelete) && (
          <div className="absolute right-1 top-1/2 -translate-y-1/2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  aria-label={`Actions for ${p.name}`}
                  className="p-0.5 rounded text-gray-300 hover:text-gray-600 hover:bg-gray-200 opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 data-[state=open]:opacity-100 data-[state=open]:bg-gray-200 data-[state=open]:text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                >
                  <MoreHorizontal className="w-4 h-4" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                {onSetStatus &&
                  (closed ? (
                    <DropdownMenuItem onSelect={() => onSetStatus(p, "active")}>
                      <PlayCircle /> Reactivate
                    </DropdownMenuItem>
                  ) : (
                    <>
                      <DropdownMenuItem
                        onSelect={() => onSetStatus(p, "completed")}
                      >
                        <CheckCircle2 /> Complete
                      </DropdownMenuItem>
                      {p.status === "on_hold" ? (
                        <DropdownMenuItem
                          onSelect={() => onSetStatus(p, "active")}
                        >
                          <PlayCircle /> Reactivate
                        </DropdownMenuItem>
                      ) : (
                        <DropdownMenuItem
                          onSelect={() => onSetStatus(p, "on_hold")}
                        >
                          <PauseCircle /> Put on hold
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuItem
                        onSelect={() => onSetStatus(p, "dropped")}
                      >
                        <XCircle /> Drop
                      </DropdownMenuItem>
                    </>
                  ))}
                {onDelete && (
                  <>
                    {onSetStatus && <DropdownMenuSeparator />}
                    <DropdownMenuItem destructive onSelect={() => onDelete(p)}>
                      <Trash2 /> Delete
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </>
    );
  };

  // Each sibling group is its own sortable list; the subtree travels with
  // its row because it is rendered inside the same sortable node.
  const group = (items: Project[], depth: number) => (
    <SortableContext
      items={items.map((p) => p.id)}
      strategy={verticalListSortingStrategy}
    >
      {items.map((p) => (
        <TreeRow
          key={p.id}
          project={p}
          depth={depth}
          canDrag={!!onReorder && items.length > 1}
          render={renderRow}
        >
          {!collapsed.has(p.id) &&
            childrenOf(p.id).length > 0 &&
            group(childrenOf(p.id), depth + 1)}
        </TreeRow>
      ))}
    </SortableContext>
  );

  return (
    <div>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        modifiers={restrictToVerticalAxisIfAvailable}
        onDragEnd={handleDragEnd}
      >
        <div className="space-y-0.5">
          {group(top, 0)}
          {orphans.length > 0 && group(orphans, 0)}
          {top.length === 0 && orphans.length === 0 && (
            <div className="text-sm text-gray-500 px-2 py-4">
              No projects yet.
            </div>
          )}
        </div>
      </DndContext>
      <label className="mt-3 flex items-center gap-2 text-xs text-gray-500 px-2">
        <Checkbox
          checked={showInactive}
          onCheckedChange={(c) => onToggleInactive(c === true)}
        />
        Show on hold, completed and dropped
      </label>
    </div>
  );
}
