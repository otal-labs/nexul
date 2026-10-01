import { z } from "zod";

export interface Doc {
  id: string;
  // Mirrors ticket.project_id; the frontend resolves the project's name from the already-fetched list.
  project_id: string;
  /** The project folder the doc lives in; every doc is in exactly one. */
  folder_id: string;
  title: string;
  body: string;
  version: number;
  archived: boolean;
  /** A locked doc refuses edits to its title and body until someone with docs:write unlocks it. */
  locked: boolean;
  /** The author's user id; "" on a doc whose creator was never recorded. */
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface DocListItem {
  id: string;
  project_id: string;
  folder_id: string;
  title: string;
  version: number;
  archived: boolean;
  locked: boolean;
  can_open: boolean;
  created_at: string;
  updated_at: string;
  /** The body's first line of text, sent only when can_open; the list search matches it. */
  snippet?: string;
}

export type DocSortField = "created_at" | "updated_at";

export const SaveDocFormSchema = z.object({
  project_id: z.string().min(1, "A project is required"),
  /** Omitted for the project's default folder. */
  folder_id: z.string().optional(),
  title: z.string().min(1, "Title is required"),
  body: z.string(),
});

export type SaveDocFormData = z.infer<typeof SaveDocFormSchema>;

export const CloneDocFormSchema = z.object({
  project_id: z.string().min(1, "A destination is required"),
});

export type CloneDocFormData = z.infer<typeof CloneDocFormSchema>;
