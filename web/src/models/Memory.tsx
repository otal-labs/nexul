import { z } from "zod";

export interface Memory {
  id: string;
  workspace_id: string;
  project_id: string;
  /** "" for an ordinary memory; "interview" for the project's interview memory, one per project. */
  kind: string;
  title: string;
  when_to_use: string;
  body: string;
  always_included: boolean;
  version: number;
  created_by: string;
  created_at: string;
  updated_by: string;
  updated_at: string;
}

// project_id "" means workspace scope; workspace_id is always sent alongside it.
export const CreateMemoryFormSchema = z.object({
  workspace_id: z.string().min(1, "A workspace is required"),
  project_id: z.string(),
  title: z.string().min(1, "Title is required"),
  when_to_use: z.string(),
  always_included: z.boolean(),
});

export type CreateMemoryFormData = z.infer<typeof CreateMemoryFormSchema>;

export const emptyCreateMemoryForm = (): CreateMemoryFormData => ({
  workspace_id: "",
  project_id: "",
  title: "",
  when_to_use: "",
  always_included: false,
});

export const isWorkspaceMemory = (memory: Memory): boolean => memory.project_id === "";

export const INTERVIEW_KIND = "interview";

// Mirrors memories.MaxInterviewChars; the server refuses a save over it.
export const MAX_INTERVIEW_CHARS = 8000;

export const isInterviewMemory = (memory: Memory): boolean => memory.kind === INTERVIEW_KIND;
