import { z } from "zod";

export const VersionLatestSchema = z.object({
  version: z.string(),
  url: z.string(),
});

export const VersionSchema = z.object({
  version: z.string(),
  channel: z.string(),
  latest: VersionLatestSchema.nullable(),
  update_available: z.boolean(),
});

export type Version = z.infer<typeof VersionSchema>;
