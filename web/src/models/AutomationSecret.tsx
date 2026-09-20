import { z } from "zod";

// The only shape any API response returns — value is write-only, never read back.
export interface AutomationSecretMeta {
  name: string;
  created_at: string;
  updated_at: string;
}

// Mirrors secrets.go's secretNamePattern exactly.
const SECRET_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;

export const SaveAutomationSecretFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name is required")
    .regex(SECRET_NAME_PATTERN, "Use letters, numbers, and underscores only, starting with a letter or underscore"),
  value: z.string().min(1, "Value is required"),
});

export type SaveAutomationSecretFormData = z.infer<typeof SaveAutomationSecretFormSchema>;
