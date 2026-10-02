import type { Project } from "@/lib/types";

/** Depth-first list of a project's descendants (depth 1 = direct child). */
export function subtree(
  projects: Project[],
  rootId: string,
): { project: Project; depth: number }[] {
  const byParent = new Map<string, Project[]>();
  for (const p of projects) {
    if (!p.parent_id) continue;
    const list = byParent.get(p.parent_id) ?? [];
    list.push(p);
    byParent.set(p.parent_id, list);
  }
  for (const list of byParent.values()) {
    list.sort(
      (a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name),
    );
  }
  const out: { project: Project; depth: number }[] = [];
  const walk = (id: string, depth: number) => {
    for (const child of byParent.get(id) ?? []) {
      out.push({ project: child, depth });
      if (depth < 3) walk(child.id, depth + 1);
    }
  };
  walk(rootId, 1);
  return out;
}
