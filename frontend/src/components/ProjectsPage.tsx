"use client";

import RequireAuth from "@/components/RequireAuth";
import ProjectTree from "@/components/ProjectTree";
import ProjectForm from "@/components/ProjectForm";
import ProjectView from "@/components/ProjectView";
import { useProjectsAndTags } from "@/hooks/useProjectsAndTags";
import { apiClient } from "@/lib/api-client";
import type { Project } from "@/lib/types";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";

function ProjectsContent({ selectedId }: { selectedId?: string }) {
  const router = useRouter();
  const {
    projects: activeProjects,
    tags,
    loaded,
    reload,
    addTag,
  } = useProjectsAndTags();
  const [allProjects, setAllProjects] = useState<Project[]>([]);
  const [showInactive, setShowInactive] = useState(false);
  const [creating, setCreating] = useState(false);

  // The picker list only holds active/on-hold projects; the tree can show
  // everything when the filter is on.
  useEffect(() => {
    (async () => {
      const { data } = await apiClient.GET("/api/projects", {
        params: { query: { status: "all" } },
      });
      if (data) setAllProjects(data);
    })();
  }, [activeProjects]);

  const onProjectChanged = useCallback(() => {
    reload();
  }, [reload]);

  const treeProjects = showInactive ? allProjects : activeProjects;

  return (
    <div className="flex flex-1 items-start">
      <div className="w-72 flex-shrink-0 p-6 md:p-8 md:pr-0 space-y-3">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">Projects</h1>
          <Button
            size="xs"
            variant="ghost"
            className="text-blue-600 hover:text-blue-700"
            onClick={() => setCreating((v) => !v)}
          >
            <Plus className="w-3.5 h-3.5" /> New
          </Button>
        </div>
        {creating && (
          <ProjectForm
            projects={activeProjects}
            onCreated={(p) => {
              setCreating(false);
              reload();
              router.push(`/projects/${p.id}`);
            }}
            onCancel={() => setCreating(false)}
          />
        )}
        <div className="bg-white border border-gray-200 rounded-lg p-2">
          {!loaded ? (
            <div className="text-sm text-gray-500 p-2">Loading…</div>
          ) : (
            <ProjectTree
              projects={treeProjects}
              selectedId={selectedId}
              showInactive={showInactive}
              onToggleInactive={setShowInactive}
            />
          )}
        </div>
      </div>
      {selectedId ? (
        <div className="flex-1 min-w-0 flex items-start">
          <ProjectView
            key={selectedId}
            projectId={selectedId}
            projects={activeProjects}
            tags={tags}
            onCreateTag={addTag}
            onProjectChanged={onProjectChanged}
            onProjectDeleted={() => {
              reload();
              router.push("/projects");
            }}
            onClose={() => router.push("/projects")}
          />
        </div>
      ) : (
        <div className="flex-1 min-w-0 p-6 md:p-8">
          <div className="bg-white border border-dashed border-gray-200 rounded-lg p-12 text-center text-sm text-gray-500">
            {loaded && activeProjects.length === 0
              ? "Create your first project to get started."
              : "Select a project to see its tasks."}
          </div>
        </div>
      )}
    </div>
  );
}

export default function ProjectsPage({ selectedId }: { selectedId?: string }) {
  return (
    <RequireAuth>
      <ProjectsContent selectedId={selectedId} />
    </RequireAuth>
  );
}
