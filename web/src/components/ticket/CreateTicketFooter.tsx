import { useEffect, useState } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { Checkbox } from "@/components/ui/checkbox";
import type { SaveTicketFormData } from "@/models/Ticket";

// Checking "create another" tells FormDialog (via context) to stay open and reset title/body instead of resolving.
export const CreateTicketFooter = () => {
  const { setStayOpen, onAfterSubmit, setValue, setFocus } = useFormDialogContext<SaveTicketFormData>();
  const [createMore, setCreateMore] = useState(false);

  useEffect(() => {
    onAfterSubmit(() => {
      setValue("title", "");
      setValue("body", "");
      setFocus("title");
    });
  }, [onAfterSubmit, setValue, setFocus]);

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
