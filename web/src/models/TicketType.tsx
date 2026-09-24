import { z } from "zod";

export interface TicketType {
  id: string;
  name: string;
  position: number;
  /** Owner-configured suggested-palette hue, "" when unset; the board falls back to ticketTypeColor.tsx's hash. */
  color: string;
  /** Markdown pre-filled into a new ticket's body; guidance only, "" when the type has none. */
  body_template: string;
  created_at: string;
  updated_at: string;
}

// Mirrors the server's rule: the type named "bug" (any case) must carry a found-in link; a renamed type is ordinary.
export const isBugType = (name: string): boolean => name.trim().toLowerCase() === "bug";

export const SaveTicketTypeFormSchema = z.object({
  name: z.string().trim().min(1, "Type name is required"),
});

export type SaveTicketTypeFormData = z.infer<typeof SaveTicketTypeFormSchema>;

export const SaveTicketTypeTemplateFormSchema = z.object({
  body_template: z.string(),
});

export type SaveTicketTypeTemplateFormData = z.infer<typeof SaveTicketTypeTemplateFormSchema>;
