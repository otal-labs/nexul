import { z } from "zod";

import { KEY_PATTERN } from "@/models/EnvFile";

const RowSchema = z.object({ key: z.string().trim(), value: z.string() });

// A row with no name and no value is a new row nobody filled in; it is dropped on save, not rejected.
export const StackEnvFormSchema = z
  .object({ rows: z.array(RowSchema) })
  .superRefine(({ rows }, ctx) => {
    const seen = new Set<string>();
    rows.forEach((row, index) => {
      const fail = (message: string) => ctx.addIssue({ code: "custom", message, path: ["rows", index, "key"] });
      if (row.key === "" && row.value === "") return;
      if (row.key === "") return fail("Name the variable.");
      if (!KEY_PATTERN.test(row.key)) return fail("Use letters, digits and underscores, and don't start with a digit.");
      if (seen.has(row.key)) return fail("Already set above.");
      seen.add(row.key);
    });
  });

export type StackEnvFormData = z.infer<typeof StackEnvFormSchema>;

export const envToRows = (env: Record<string, string> | undefined): StackEnvFormData["rows"] =>
  Object.entries(env ?? {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, value]) => ({ key, value }));

export const rowsToEnv = (rows: StackEnvFormData["rows"]): Record<string, string> =>
  Object.fromEntries(rows.filter((row) => row.key !== "").map((row) => [row.key, row.value]));
