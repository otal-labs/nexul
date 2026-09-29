import { z } from "zod";

export interface Doc {
  id: string;
  // Mirrors ticket.project_id; the frontend resolves the project's name from the already-fetched list.
  project_id: string;
  title: string;
  body: string;
  version: number;
  archived: boolean;
  /** The author's user id; "" on a doc whose creator was never recorded. */
  created_by: string;
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
  /** The author and the body's first line of text, sent only when can_open. */
  created_by?: string;
  snippet?: string;
}

export const SaveDocFormSchema = z.object({
  project_id: z.string().min(1, "A project is required"),
  title: z.string().min(1, "Title is required"),
  body: z.string(),
});

export type SaveDocFormData = z.infer<typeof SaveDocFormSchema>;

export const CloneDocFormSchema = z.object({
  project_id: z.string().min(1, "A destination is required"),
});

export type CloneDocFormData = z.infer<typeof CloneDocFormSchema>;
