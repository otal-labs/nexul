import { z } from "zod";

/** A project's group of docs, at the top or inside another folder; the default one holds new docs and is never deleted or moved. */
export interface DocFolder {
  id: string;
  project_id: string;
  /** The folder it sits inside; empty at the top level. */
  parent_id: string;
  name: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export const DocFolderFormSchema = z.object({
  name: z.string().trim().min(1, "A name is required"),
});

export type DocFolderFormData = z.infer<typeof DocFolderFormSchema>;
