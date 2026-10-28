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

// Only http(s) reaches a link or an image; a pattern, not URL, since React Native's URL parser accepts anything.
const HTTP_URL = /^https?:\/\/[^\s/?#]+[^\s]*$/i;

export const httpUrl = (raw: string | undefined): string | undefined => (raw && HTTP_URL.test(raw) ? raw : undefined);

// A sender's image as the media proxy serves it, so the sender's host never sees who reads the message.
export const botMediaPath = (messageId: string, raw: string | undefined): string | undefined => {
  const url = httpUrl(raw);
  if (!url) return undefined;
  return `/api/botwebhooks/media?message=${encodeURIComponent(messageId)}&url=${encodeURIComponent(url)}`;
};

const ZONELESS_TIME = /^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}(:\d{2}(\.\d+)?)?$/;

// Discord reads a timestamp without a zone as UTC; Uptime Kuma sends one, and the client would read it as local.
export const embedTimestamp = (raw: string): string => (ZONELESS_TIME.test(raw) ? `${raw.replace(" ", "T")}Z` : raw);

export type EmbedTone = "success" | "warning" | "destructive" | "info";

// First match wins, worst first: "failed, rolled back to the healthy image" reads as a failure.
const TONE_WORDS: [EmbedTone, RegExp][] = [
  ["destructive", /\b(fail(s|ed|ing|ure)?|errors?|errored|crash(ed|ing)?|outage|broken|critical|rejected)\b|\[down\]|\b(is|went) down\b/i],
  ["warning", /\b(warn(s|ing)?|degraded|pending|stopped|paused|retrying|timed out|unstable|skipped)\b/i],
  ["success", /\b(healthy|succeeded|success(ful)?|passed|resolved|recovered|ok|completed?|finished|deployed)\b|\[up\]|\b(is|back) up\b/i],
  ["info", /\b(started|running|in progress|queued|redeployed|redeploying|deploying|building)\b/i],
];

// The server drops a sender's color, so a post's state is read from its words; null when they name none.
// ponytail: an English keyword list; a sender in another language reads as neutral.
export const embedTone = (text: string | undefined): EmbedTone | null =>
  (text && TONE_WORDS.find(([, words]) => words.test(text))?.[0]) || null;

// The card's state comes from its headline, the description only when there is no title: "Nightly summary" stays neutral.
export const embedCardTone = (embed: Embed): EmbedTone | null => embedTone(embed.title ?? embed.description);

// A field whose value is a state word or two ("healthy", "timed out") gets a status dot; a sentence stays plain.
export const fieldTone = (value: string): EmbedTone | null => (value.split(/\s+/).length <= 3 ? embedTone(value) : null);

// One entry of a report's state tally: a problem by its field's name, or a count of the fields in one state.
export interface EmbedTallyEntry {
  tone: EmbedTone;
  value: string;
  name?: string;
  count?: number;
}

export interface EmbedFacts {
  line: EmbedField[];
  blocks: EmbedField[];
  tally: EmbedTallyEntry[];
  folded: number;
}

const TALLY_MIN = 3;
const SHORT_FACT = 40;
const TONE_RANK: Record<EmbedTone, number> = { destructive: 0, warning: 1, info: 2, success: 3 };

// Problems by name, worst first, then the rest counted by their state word: "cache stopped · 8 healthy".
const tallyOf = (stated: { field: EmbedField; tone: EmbedTone }[]): EmbedTallyEntry[] => {
  const ranked = [...stated].sort((a, b) => TONE_RANK[a.tone] - TONE_RANK[b.tone]);
  const problems = ranked
    .filter((s) => s.tone === "destructive" || s.tone === "warning")
    .map(({ field, tone }) => ({ tone, name: field.name, value: field.value }));
  const counts = new Map<string, EmbedTallyEntry>();
  for (const { field, tone } of ranked) {
    if (tone === "destructive" || tone === "warning") continue;
    const key = field.value.toLowerCase();
    const entry = counts.get(key) ?? { tone, value: key, count: 0 };
    counts.set(key, { ...entry, count: (entry.count ?? 0) + 1 });
  }
  return [...problems, ...counts.values()];
};

// A closed report's facts: short ones on one line, long ones a line each, three or more state fields as a tally.
export const embedFacts = (fields: EmbedField[] = []): EmbedFacts => {
  const toned = fields.map((field) => ({ field, tone: fieldTone(field.value) }));
  const stated = toned.flatMap(({ field, tone }) => (tone ? [{ field, tone }] : []));
  const tallied = stated.length >= TALLY_MIN;
  const rest = tallied ? toned.filter((t) => !t.tone).map((t) => t.field) : fields;
  const line = rest.filter((f) => f.inline && f.value.length <= SHORT_FACT && !f.value.includes("\n"));
  const blocks = rest.filter((f) => !line.includes(f));
  return {
    line,
    blocks: blocks.slice(0, EMBED_FIELD_LIMIT),
    tally: tallied ? tallyOf(stated) : [],
    folded: Math.max(0, blocks.length - EMBED_FIELD_LIMIT),
  };
};

// The report's fold label while closed: a tally opens every field, else what the closed report hides; null for nothing.
export const embedFactsFoldLabel = (embed: Embed, facts: EmbedFacts): string | null => {
  if (facts.tally.length > 0) return `Show all ${embed.fields?.length ?? 0} fields`;
  if (facts.folded > 0) return `Show ${plural(facts.folded, "field")}`;
  return embedFoldLabel({ hiddenFields: 0, longDescription: embedFold(embed).longDescription });
};

// An author line that only names the bot posting it repeats the message header above the card.
export const dropEchoedAuthor = (embed: Embed, botName: string): Embed => {
  if (embed.author?.name !== botName || embed.author.url || embed.author.icon_url) return embed;
  const rest = { ...embed };
  delete rest.author;
  return rest;
};
