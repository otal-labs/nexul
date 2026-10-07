import { z } from "zod";

import type { ConversationKind } from "@/models/Chat";
import { formatRelativeTime } from "@/utils/TimeUtility";

// A conversation's poster, driven from outside through its webhook URL (internal/botwebhook/model.go).
export interface Botwebhook {
  id: string;
  conversation_id: string;
  name: string;
  // A base64 data URL; absent shows the Nexul glyph.
  avatar?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  last_post_at?: string;
  post_count: number;
  deleted_at?: string;
  deleted_by?: string;
  // Only for a viewer holding botwebhook:write.
  url?: string;
}

// Mirrors the server's MaxLive and MaxNameLength.
export const BOT_CAP = 10;
const MAX_NAME_LENGTH = 80;

// Names are unique per conversation among live bots, case-insensitively, so taken holds the other live bots' names.
export const botFormSchema = (taken: string[]) =>
  z.object({
    name: z
      .string()
      .trim()
      .min(1, "Give the bot a name.")
      .max(MAX_NAME_LENGTH, "Keep the name to 80 characters.")
      .refine((name) => !taken.some((other) => other.toLowerCase() === name.toLowerCase()), {
        error: (issue) => `Another bot here is called ${String(issue.input).trim()}.`,
      }),
    avatar: z.string(),
  });

export type BotFormData = z.infer<ReturnType<typeof botFormSchema>>;

export const capNote = (kind: ConversationKind): string =>
  `A ${kind === "channel" || kind === "voice_channel" ? "channel" : "conversation"} holds ten bots. Delete one to add another.`;

export const botActivityLine = (bot: Botwebhook): string =>
  bot.last_post_at ? `Last post ${formatRelativeTime(bot.last_post_at)}` : "No posts yet";
