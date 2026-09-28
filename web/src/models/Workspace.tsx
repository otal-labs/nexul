import { z } from "zod";

// Mirrors internal/tenancy/model.go's Workspace wire shape (ticket 07).
export interface Workspace {
  id: string;
  name: string;
  // A free-text template with {ticket.Field} placeholders; any member reads it, only workspaces:write can change it.
  mention_chip_template: string;
  created_at: string;
  updated_at: string;
}

// Matches the column default on workspaces so a chip looks the same before the workspace list resolves.
export const DEFAULT_MENTION_CHIP_TEMPLATE = "{ticket.Ticket} {ticket.Status}";

export const SaveWorkspaceFormSchema = z.object({
  name: z.string().trim().min(1, "Workspace name is required"),
});

export type SaveWorkspaceFormData = z.infer<typeof SaveWorkspaceFormSchema>;
