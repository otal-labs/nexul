import { ChevronRightIcon } from "lucide-react";
import { useState } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { menuItemClass, pillTriggerClass } from "@/components/ticket/ticketFormPillStyles";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { SaveTicketFormData } from "@/models/Ticket";

// Shares CreateTicketForm's form instance, so picking a project here drives its re-seed effect.
export const CreateTicketHeader = () => {
  const { watch, setValue } = useFormDialogContext<SaveTicketFormData>();
  const { data: projects } = useFetchProjects();
  const [open, setOpen] = useState(false);
  const projectId = watch("project_id");
  const project = (projects ?? []).find((p) => p.id === projectId);

  return (
    <div className="flex items-center gap-1.5">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button type="button" className={pillTriggerClass}>
            {project?.name ?? "Project"}
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-56 p-1">
          <div className="flex flex-col gap-0.5">
            {(projects ?? []).map((p) => (
              <button
                key={p.id}
                type="button"
                className={menuItemClass}
                onClick={() => {
                  setValue("project_id", p.id);
                  setOpen(false);
                }}
              >
                {p.name}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
      <ChevronRightIcon className="size-3.5 text-muted-foreground" aria-hidden />
      <span className="text-sm font-medium text-foreground">New ticket</span>
    </div>
  );
};
