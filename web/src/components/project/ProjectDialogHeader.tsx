import { ChevronRightIcon } from "lucide-react";
import type { ReactNode } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { PillPicker } from "@/components/PillPicker";
import { useFetchProjects } from "@/hooks/ProjectHooks";

interface ProjectDialogHeaderProps {
  title: string;
  /** Pills between the project and the title, each followed by its own chevron. */
  children?: ReactNode;
}

// A create dialog's project pill and title; it shares the form's instance, so a pick here drives the form's re-seed.
export const ProjectDialogHeader = ({ title, children }: ProjectDialogHeaderProps) => {
  const { watch, setValue } = useFormDialogContext<{ project_id: string }>();
  const { data: projects } = useFetchProjects();
  const projectId = watch("project_id");
  const project = (projects ?? []).find((p) => p.id === projectId);

  return (
    <div className="flex items-center gap-1.5">
      <PillPicker label={project?.name ?? "Project"} items={projects ?? []} onPick={(id) => setValue("project_id", id)} />
      <ChevronRightIcon className="size-3.5 text-muted-foreground" aria-hidden />
      {children}
      <span className="text-sm font-medium text-foreground">{title}</span>
    </div>
  );
};
