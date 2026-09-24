import type { UseFormReturn } from "react-hook-form";

import type { SaveTicketFormData } from "@/models/Ticket";
import type { TicketType } from "@/models/TicketType";

export type TicketTypeForm = Pick<UseFormReturn<SaveTicketFormData>, "setValue" | "getValues">;

// The body follows the type only while it is empty or still an unedited template, so typed text is never replaced.
export const bodyForType = (body: string, ticketTypes: TicketType[], typeId: string): string => {
  const untouched = body.trim() === "" || ticketTypes.some((t) => t.body_template === body);
  if (!untouched) return body;
  return ticketTypes.find((t) => t.id === typeId)?.body_template ?? "";
};

export const selectTicketType = (form: TicketTypeForm, ticketTypes: TicketType[], typeId: string) => {
  form.setValue("type_id", typeId);
  form.setValue("body", bodyForType(form.getValues("body"), ticketTypes, typeId));
};
