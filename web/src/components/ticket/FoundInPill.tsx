import { Bug } from "lucide-react";
import { useState } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { TicketPickerList } from "@/components/ticket/TicketPickerList";
import { pillTriggerClass } from "@/components/ticket/ticketFormPillStyles";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTickets } from "@/hooks/TicketHooks";
import type { SaveTicketFormData } from "@/models/Ticket";

interface FoundInPillProps {
  allowUnknown: boolean;
}

// The bug's origin: a picked ticket, or, where allowed, "origin unknown" recorded instead of a guessed link.
export const FoundInPill = ({ allowUnknown }: FoundInPillProps) => {
  const { watch, setValue, formState } = useFormDialogContext<SaveTicketFormData>();
  const { data: tickets } = useFetchTickets();
  const { data: projects } = useFetchProjects();
  const [open, setOpen] = useState(false);
  const originId = watch("origin_id") ?? "";
  const unknown = watch("origin_unknown") === true;
  const origin = (tickets ?? []).find((t) => t.id === originId);
  const prefix = (projects ?? []).find((p) => p.id === origin?.project_id)?.prefix;
  const label = origin ? `Found in ${prefix ? `${prefix}-${origin.number}` : origin.title}` : "Found in…";
  const error = formState.errors.origin_id?.message as string | undefined;

  return (
    <span className="contents">
      {!unknown && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <button type="button" className={pillTriggerClass} aria-invalid={error != null}>
              <Bug className="size-3.5 text-muted-foreground" aria-hidden />
              <span className="max-w-40 truncate">{label}</span>
            </button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-72 max-w-[calc(100vw-2rem)] p-1.5">
            <TicketPickerList
              excludeId=""
              onSelect={(id) => {
                setValue("origin_id", id, { shouldValidate: true });
                setOpen(false);
              }}
            />
          </PopoverContent>
        </Popover>
      )}
      {allowUnknown && (
        <label className="inline-flex h-7 items-center gap-1.5 px-1 text-xs text-muted-foreground">
          <Checkbox
            checked={unknown}
            onCheckedChange={(checked) => {
              setValue("origin_unknown", checked === true);
              setValue("origin_id", "", { shouldValidate: true });
            }}
          />
          Origin unknown
        </label>
      )}
      {error && (
        <p role="alert" className="basis-full text-sm text-destructive">
          {error}
        </p>
      )}
    </span>
  );
};
