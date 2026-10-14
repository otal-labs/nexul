import type { ComponentProps } from "react";

import { ActionInlineEdit } from "@/components/action/ActionInlineEdit";
import { editableRowClass, liveRowClass } from "@/components/ticket/ticketPropertyRowStyle";

interface PropertyRowTriggerProps extends Omit<ComponentProps<typeof ActionInlineEdit>, "permission" | "className" | "liveClassName" | "pencil"> {
  projectId: string;
}

// A property row of the ticket's rail: a popover trigger that needs tickets:write, and plain text without it.
export const PropertyRowTrigger = (props: PropertyRowTriggerProps) => (
  <ActionInlineEdit permission="tickets:write" pencil={false} className={editableRowClass} liveClassName={liveRowClass} {...props} />
);
