// A bot message's embed as the sender posted it in Discord's execute-webhook JSON; the server drops its color.
export interface EmbedField {
  name: string;
  value: string;
  inline?: boolean;
}

export interface Embed {
  title?: string;
  description?: string;
  url?: string;
  timestamp?: string;
  footer?: { text: string; icon_url?: string };
  image?: { url: string };
  thumbnail?: { url: string };
  author?: { name: string; url?: string; icon_url?: string };
  fields?: EmbedField[];
}

export const EMBED_FIELD_LIMIT = 6;
export const EMBED_STACK_LIMIT = 2;
const LONG_DESCRIPTION_CHARS = 420;
const LONG_DESCRIPTION_LINES = 8;

export interface EmbedFold {
  hiddenFields: number;
  longDescription: boolean;
}

// What a closed embed hides: the fields past six, and a description long enough to clamp.
export const embedFold = (embed: Embed): EmbedFold => {
  const description = embed.description ?? "";
  return {
    hiddenFields: Math.max(0, (embed.fields?.length ?? 0) - EMBED_FIELD_LIMIT),
    longDescription: description.length > LONG_DESCRIPTION_CHARS || description.split("\n").length > LONG_DESCRIPTION_LINES,
  };
};

const plural = (n: number, word: string) => `${n} more ${word}${n === 1 ? "" : "s"}`;

// The fold bar's label while closed; null when there is nothing to fold.
export const embedFoldLabel = ({ hiddenFields, longDescription }: EmbedFold): string | null => {
  if (hiddenFields > 0) return `Show ${plural(hiddenFields, "field")}`;
  if (longDescription) return "Show the rest";
  return null;
};

export const moreEmbedsLabel = (hidden: number): string => `Show ${plural(hidden, "embed")}`;

// Only http(s) reaches an href or an <img src>, so a javascript: or data: URL from a sender stays inert.
export const httpUrl = (raw: string | undefined): string | undefined => {
  if (!raw) return undefined;
  try {
    const url = new URL(raw);
    return url.protocol === "http:" || url.protocol === "https:" ? raw : undefined;
  } catch {
    return undefined;
  }
};

// A sender's image as the media proxy serves it, so the sender's host never sees who reads the message.
export const botMediaPath = (messageId: string, raw: string | undefined): string | undefined => {
  const url = httpUrl(raw);
  if (!url) return undefined;
  return `/api/botwebhooks/media?message=${encodeURIComponent(messageId)}&url=${encodeURIComponent(url)}`;
};

const ZONELESS_TIME = /^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}(:\d{2}(\.\d+)?)?$/;

// Discord reads a timestamp without a zone as UTC; Uptime Kuma sends one, and the browser would read it as local.
export const embedTimestamp = (raw: string): string => (ZONELESS_TIME.test(raw) ? `${raw.replace(" ", "T")}Z` : raw);
