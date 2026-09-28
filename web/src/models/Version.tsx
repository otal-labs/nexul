import { z } from "zod";

export const VersionLatestSchema = z.object({
  version: z.string(),
  url: z.string(),
});

export const VersionChangeSchema = z.object({
  version: z.string(),
  url: z.string(),
  notes: z.array(z.string()),
});

export const VersionSchema = z.object({
  version: z.string(),
  channel: z.string(),
  latest: VersionLatestSchema.nullable(),
  update_available: z.boolean(),
  changes: z.array(VersionChangeSchema),
});

export type Version = z.infer<typeof VersionSchema>;
export type VersionLatest = z.infer<typeof VersionLatestSchema>;
export type VersionChange = z.infer<typeof VersionChangeSchema>;
