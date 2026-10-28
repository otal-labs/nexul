import type { ReactNode } from "react";

import { fieldTone, type EmbedFacts, type EmbedField, type EmbedTallyEntry } from "@nexul/client-core/embed";

import { DiscordInline, DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { toneBg } from "@/components/chat/EmbedTone";
import { cn } from "@/lib/utils";

const Dot = ({ field, tone }: { field?: EmbedField; tone?: EmbedTallyEntry["tone"] | null }) => {
  const hue = tone ?? (field && fieldTone(field.value));
  if (!hue) return null;
  return <span aria-hidden className={cn("size-1.5 shrink-0 translate-y-[-1px] self-center rounded-full", toneBg[hue])} />;
};

const Separated = ({ children, first }: { children: ReactNode; first: boolean }) => (
  <span className="inline-flex min-w-0 items-baseline gap-1.5">
    {!first && <span aria-hidden className="text-muted-foreground/60">·</span>}
    {children}
  </span>
);

const FactLine = ({ fields }: { fields: EmbedField[] }) => (
  <p className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-[13px] leading-5">
    {fields.map((field, i) => (
      <Separated key={i} first={i === 0}>
        <span className="text-muted-foreground">{field.name}</span>
        <Dot field={field} />
        <DiscordInline text={field.value} className="font-mono text-xs text-foreground" />
      </Separated>
    ))}
  </p>
);

const TallyLine = ({ tally }: { tally: EmbedTallyEntry[] }) => (
  <p className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-[13px] leading-5">
    {tally.map((entry, i) => (
      <Separated key={i} first={i === 0}>
        <Dot tone={entry.tone} />
        {entry.name && <span className="font-mono text-xs text-foreground">{entry.name}</span>}
        {entry.count !== undefined && <span className="font-mono text-xs text-foreground tabular-nums">{entry.count}</span>}
        <span className={cn(entry.name ? "text-foreground" : "text-muted-foreground")}>{entry.value}</span>
      </Separated>
    ))}
  </p>
);

const FactBlock = ({ field }: { field: EmbedField }) => (
  <div className="flex min-w-0 items-baseline gap-2 text-[13px] leading-5">
    <span className="shrink-0 text-muted-foreground">{field.name}</span>
    <Dot field={field} />
    <DiscordMarkdown text={field.value} className="min-w-0 text-foreground" />
  </div>
);

// A closed report's facts as sentences: the state tally, the short facts on one line, then each long fact on its own.
export const EmbedFactLines = ({ facts }: { facts: EmbedFacts }) => (
  <div className="space-y-1">
    {facts.tally.length > 0 && <TallyLine tally={facts.tally} />}
    {facts.line.length > 0 && <FactLine fields={facts.line} />}
    {facts.blocks.map((field, i) => (
      <FactBlock key={i} field={field} />
    ))}
  </div>
);
