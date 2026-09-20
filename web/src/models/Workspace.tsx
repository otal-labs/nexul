import { z } from "zod";

// Mirrors internal/tenancy/model.go's Workspace wire shape (ticket 07).
export interface Workspace {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export const SaveWorkspaceFormSchema = z.object({
  name: z.string().trim().min(1, "Workspace name is required"),
});

export type SaveWorkspaceFormData = z.infer<typeof SaveWorkspaceFormSchema>;
