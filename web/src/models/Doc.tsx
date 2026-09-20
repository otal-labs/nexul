import { z } from "zod";

export interface Doc {
  id: string;
  // Mirrors ticket.project_id; the frontend resolves the project's name from the already-fetched list.
  project_id: string;
  title: string;
  body: string;
  version: number;
  archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface DocListItem {
  id: string;
  project_id: string;
  title: string;
  version: number;
  archived: boolean;
  can_open: boolean;
  updated_at: string;
}

export const SaveDocFormSchema = z.object({
  project_id: z.string().min(1, "A project is required"),
  title: z.string().min(1, "Title is required"),
  body: z.string(),
});

export type SaveDocFormData = z.infer<typeof SaveDocFormSchema>;
