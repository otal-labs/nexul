import { z } from "zod";

export interface Category {
  id: string;
  project_id: string;
  name: string;
  position: number;
  /** Owner-configured suggested-palette hue, "" when unset — same shape as TicketType.color. */
  color: string;
  created_at: string;
  updated_at: string;
}

export const SaveCategoryFormSchema = z.object({
  project_id: z.string().trim().min(1, "A project is required"),
  name: z.string().trim().min(1, "Category name is required"),
  color: z.string(),
});

export type SaveCategoryFormData = z.infer<typeof SaveCategoryFormSchema>;
