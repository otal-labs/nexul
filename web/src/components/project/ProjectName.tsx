import { useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { ActionInlineEdit } from "@/components/action/ActionInlineEdit";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useEntityActions } from "@/hooks/useEntityActions";
import type { Project } from "@/models/Project";

interface ProjectNameProps {
  project: Project;
  onRename: (name: string) => Promise<unknown>;
  className?: string;
}

export const ProjectName = ({ project, onRename, className }: ProjectNameProps) => {
  const [editing, setEditing] = useState(false);
  const answer = useEntityActions("project", project.id);
  const form = useForm<{ name: string }>({ defaultValues: { name: project.name } });

  const saveRename = () => {
    void form.handleSubmit(async ({ name }) => {
      const trimmed = name.trim();
      if (trimmed && trimmed !== project.name) {
        await onRename(trimmed);
      }
      setEditing(false);
    })();
  };

  return (
    <>
      {editing && (
        <Controller
          control={form.control}
          name="name"
          render={({ field }) => (
            <Input
              id="project-name"
              className="flex-1"
              aria-label="Project name"
              autoFocus
              {...field}
              onBlur={saveRename}
              onKeyDown={(event) => {
                if (event.key === "Enter") saveRename();
                if (event.key === "Escape") {
                  form.reset({ name: project.name });
                  setEditing(false);
                }
              }}
            />
          )}
        />
      )}
      {!editing && (
        <ActionInlineEdit
          permission="projects:write"
          projectId={project.id}
          answer={answer("projects:write")}
          label="Rename project"
          onEdit={() => setEditing(true)}
          className={cn("flex-1 font-medium", className)}
        >
          {project.name}
        </ActionInlineEdit>
      )}
    </>
  );
};
