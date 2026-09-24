import type { UseFormReturn } from "react-hook-form";

import type { SaveTicketFormData } from "@/models/Ticket";
import { isBugType, type TicketType } from "@/models/TicketType";

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

// Bugs are filed through Report a bug, so the normal dialog offers every other type; a project with no bug type offers all.
export const offeredTicketTypes = (ticketTypes: TicketType[], bug: boolean): TicketType[] => {
  if (!bug) return ticketTypes.filter((t) => !isBugType(t.name));
  const bugTypes = ticketTypes.filter((t) => isBugType(t.name));
  return bugTypes.length > 0 ? bugTypes : ticketTypes;
};
