import { useEffect, useState } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { bodyForType } from "@/components/ticket/selectTicketType";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import type { SaveTicketFormData } from "@/models/Ticket";

// Checking "create another" tells FormDialog (via context) to stay open and reset title/body instead of resolving.
export const CreateTicketFooter = () => {
  const { setStayOpen, onAfterSubmit, setValue, getValues, setFocus, watch } = useFormDialogContext<SaveTicketFormData>();
  const [createMore, setCreateMore] = useState(false);
  const { data: ticketTypes } = useFetchProjectTicketTypes(watch("project_id"));

  useEffect(() => {
    onAfterSubmit(() => {
      setValue("title", "");
      setValue("body", bodyForType("", ticketTypes ?? [], getValues("type_id") ?? ""));
      setFocus("title");
    });
  }, [onAfterSubmit, setValue, getValues, setFocus, ticketTypes]);

  return (
    <label className="flex items-center gap-2 text-xs text-muted-foreground">
      <Checkbox
        checked={createMore}
        onCheckedChange={(checked) => {
          const value = checked === true;
          setCreateMore(value);
          setStayOpen(value);
        }}
      />
      Create more
    </label>
  );
};
