import { z } from "zod";

export const AuditMinDays = 45;
export const AuditMaxDays = 1095;

// What the Owner reads back; the stream secret is never sent, only whether one is stored, except new_secret once.
export const AuditSettingsSchema = z.object({
  unlocked: z.boolean(),
  retention_days: z.number(),
  stream_url: z.string(),
  stream_secret_set: z.boolean(),
  stream_enabled: z.boolean(),
  streaming: z.boolean(),
  pending: z.number(),
  dead: z.number(),
  last_delivered_at: z.string().optional(),
  last_error: z.string().optional(),
  new_secret: z.string().optional(),
});

export type AuditSettings = z.infer<typeof AuditSettingsSchema>;

export interface AuditSettingsPatch {
  retention_days?: number;
  stream_url?: string;
  stream_secret?: string;
  stream_enabled?: boolean;
}

// What a test delivery got back: the receiver's status, or why nothing came back.
export interface AuditStreamTest {
  status: number;
  error?: string;
}

const isStreamURL = (raw: string) => {
  if (raw === "") return true;
  try {
    const u = new URL(raw);
    if (u.username !== "" || u.password !== "") return false;
    if (u.protocol === "https:") return true;
    return u.protocol === "http:" && ["localhost", "127.0.0.1", "[::1]"].includes(u.hostname);
  } catch {
    return false;
  }
};

// Days stay text in the form, so an emptied field reads as wrong rather than as zero.
export const AuditFormSchema = z
  .object({
    retention_days: z
      .string()
      .trim()
      .regex(/^\d+$/, "Enter a whole number of days")
      .refine((v) => Number(v) >= AuditMinDays && Number(v) <= AuditMaxDays, `From ${AuditMinDays} to ${AuditMaxDays.toLocaleString("en")} days`),
    stream_url: z.string().trim().refine(isStreamURL, "An https address, with no user or password in it"),
    stream_secret: z.string().trim(),
    stream_enabled: z.boolean(),
  })
  .refine((v) => !v.stream_enabled || v.stream_url !== "", { message: "Add a destination before sending", path: ["stream_enabled"] });

export type AuditFormData = z.infer<typeof AuditFormSchema>;

export const auditFormValues = (s: AuditSettings): AuditFormData => ({
  retention_days: String(s.retention_days),
  stream_url: s.stream_url,
  stream_secret: "",
  stream_enabled: s.stream_enabled,
});

// The patch a save sends: a blank secret keeps the saved one, or lets the server make one for a new destination.
export const auditPatchOf = (data: AuditFormData): AuditSettingsPatch => ({
  retention_days: Number(data.retention_days),
  stream_url: data.stream_url,
  stream_enabled: data.stream_enabled,
  ...(data.stream_secret !== "" && { stream_secret: data.stream_secret }),
});
