"use client";

import ProjectsPage from "@/components/ProjectsPage";
import { useParams } from "next/navigation";

export default function Page() {
  const params = useParams<{ id: string }>();
  return <ProjectsPage selectedId={params.id} />;
}
