import { PlusIcon } from "lucide-react";
import { Suspense } from "react";

import { LazyCreateDocForm } from "@/components/doc/LazyCreateDocForm";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFormDialog } from "@/hooks/useFormDialog";
import type { SaveDocFormData } from "@/models/Doc";
import { SaveDocFormSchema } from "@/models/Doc";
import type { Project } from "@/models/Project";
import { emptyDocForm } from "@/utils/emptyDocJson";

interface NewDocButtonProps {
  project: Project;
}

// Revealed by hovering the project switcher row (group/project).
export const NewDocButton = ({ project }: NewDocButtonProps) => {
  const { open } = useFormDialog();

  const createDoc = () =>
    open<SaveDocFormData>({
      title: "New doc",
      schema: SaveDocFormSchema,
      okLabel: "Create",
      form: (
        <Suspense fallback={<LoadingDisplay />}>
          <LazyCreateDocForm defaultProjectId={project.id} />
        </Suspense>
      ),
      formOptions: { defaultValues: { ...emptyDocForm(), project_id: project.id } },
    });

  return (
    <button
      type="button"
      onClick={() => void createDoc()}
      aria-label={`New doc in ${project.name}`}
      className="shrink-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:opacity-100 group-hover/project:opacity-100"
    >
      <PlusIcon className="size-3.5" aria-hidden />
    </button>
  );
};
