import { PlusIcon } from "lucide-react";
import { Suspense, useState } from "react";
import { useNavigate } from "react-router";

import { LazyCreateDocForm } from "@/components/doc/LazyCreateDocForm";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { navLinkClass, sectionLabelClass } from "@/components/SidebarNav";
import { ProjectTreeItem } from "@/components/sidebar/ProjectTreeItem";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { cn } from "@/lib/utils";
import type { SaveDocFormData } from "@/models/Doc";
import { SaveDocFormSchema } from "@/models/Doc";
import { emptyDocForm } from "@/utils/emptyDocJson";

interface ProjectTreeProps {
  collapsed: boolean;
}

// One openIds Set is shared by the collapsed rail and the expanded tree, so opening one keeps the other open.
export const ProjectTree = ({ collapsed }: ProjectTreeProps) => {
  const { data: projects = [] } = useFetchProjects();
  const [openIds, setOpenIds] = useState<Set<string>>(new Set());
  const navigate = useNavigate();
  const { open: openCreateDoc } = useFormDialog();

  const toggle = (id: string) =>
    setOpenIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
        return next;
      }
      next.add(id);
      return next;
    });

  // The project wizard's project step covers name + prefix (spec §1, door 1); no dialog needed here anymore.
  const createProject = () => navigate("/wizard/project/project");

  // Hovering a row reveals a "+" that adds a child, scoped here to project->doc.
  const createDoc = (projectId: string) =>
    openCreateDoc<SaveDocFormData>({
      title: "New doc",
      schema: SaveDocFormSchema,
      okLabel: "Create",
      form: (
        <Suspense fallback={<LoadingDisplay />}>
          <LazyCreateDocForm defaultProjectId={projectId} />
        </Suspense>
      ),
      formOptions: { defaultValues: { ...emptyDocForm(), project_id: projectId } },
    });

  return (
    <div className="flex flex-col gap-0.5">
      {!collapsed && (
        <div className={cn("group/projects flex items-center justify-between", sectionLabelClass)}>
          <span>Projects</span>
          <button
            type="button"
            onClick={createProject}
            aria-label="New project"
            className="rounded p-0.5 text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:opacity-100 group-hover/projects:opacity-100"
          >
            <PlusIcon className="size-3.5" aria-hidden />
          </button>
        </div>
      )}
      {projects.map((project) => (
        <ProjectTreeItem
          key={project.id}
          project={project}
          collapsed={collapsed}
          open={openIds.has(project.id)}
          onToggle={() => toggle(project.id)}
          onCreateDoc={() => void createDoc(project.id)}
        />
      ))}
      {collapsed && (
        <button
          type="button"
          onClick={createProject}
          title="New project"
          className={cn(navLinkClass({ isActive: false }), "w-full justify-center text-muted-foreground")}
        >
          <PlusIcon className="size-4 shrink-0" aria-hidden />
        </button>
      )}
    </div>
  );
};
