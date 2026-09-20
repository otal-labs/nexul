import { z } from "zod";

export interface TicketType {
  id: string;
  name: string;
  position: number;
  /** Owner-configured suggested-palette hue, "" when unset; the board falls back to ticketTypeColor.tsx's hash. */
  color: string;
  created_at: string;
  updated_at: string;
}

export const SaveTicketTypeFormSchema = z.object({
  name: z.string().trim().min(1, "Type name is required"),
});

export type SaveTicketTypeFormData = z.infer<typeof SaveTicketTypeFormSchema>;
