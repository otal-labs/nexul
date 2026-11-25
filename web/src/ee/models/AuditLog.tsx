import { z } from "zod";

// What the Owner reads back; the stream secret is never sent, only whether one is stored, except new_secret once.
export const AuditSettingsSchema = z.object({
  unlocked: z.boolean(),
  retention_days: z.number(),
  stream_url: z.string(),
  stream_secret_set: z.boolean(),
  stream_enabled: z.boolean(),
  streaming: z.boolean(),
  new_secret: z.string().optional(),
});

export type AuditSettings = z.infer<typeof AuditSettingsSchema>;
