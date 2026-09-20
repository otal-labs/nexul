import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { PencilIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Project } from "@/models/Project";

interface ProjectNameProps {
  project: Project;
  onRename: (name: string) => Promise<unknown>;
  className?: string;
}

export const ProjectName = ({ project, onRename, className }: ProjectNameProps) => {
  const [editing, setEditing] = useState(false);
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
        <>
          <span className="flex-1 font-medium">{project.name}</span>
          <Button
            variant="ghost"
            size="sm"
            aria-label="Rename project"
            className={className}
            onClick={() => setEditing(true)}
          >
            <PencilIcon className="size-4" />
          </Button>
        </>
      )}
    </>
  );
};
