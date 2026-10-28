import { useState } from "react";

import { embedCardTone, embedFacts, embedFactsFoldLabel, embedFold, type Embed } from "@nexul/client-core/embed";

import { BotMediaImage } from "@/components/chat/BotMediaImage";
import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedFactLines } from "@/components/chat/EmbedFactLines";
import { EmbedFieldGrid } from "@/components/chat/EmbedFieldGrid";
import { EmbedFooter } from "@/components/chat/EmbedFooter";
import { EmbedHeading } from "@/components/chat/EmbedHeading";
import { toneBg } from "@/components/chat/EmbedTone";
import { FoldBar } from "@/components/chat/FoldBar";
import { cn } from "@/lib/utils";

// One embed as a report on the panel, its state on the leading edge; opened, its facts lay out as the sender's grid.
export const EmbedCard = ({ embed }: { embed: Embed }) => {
  const [open, setOpen] = useState(false);
  const facts = embedFacts(embed.fields);
  const foldLabel = embedFactsFoldLabel(embed, facts);
  const tone = embedCardTone(embed);
  const hasFooter = !!(embed.footer || embed.timestamp);
  const clamp = embedFold(embed).longDescription && !open;

  return (
    <article className="@container relative pl-3">
      <span aria-hidden className={cn("absolute inset-y-0.5 left-0 w-[3px] rounded-full", tone ? toneBg[tone] : "bg-border")} />
      <div className="flex flex-col-reverse gap-3 @[22rem]:flex-row">
        <div className="min-w-0 flex-1 space-y-1">
          <EmbedHeading embed={embed} tone={tone} />
          {embed.description && (
            <DiscordMarkdown text={embed.description} className={cn("space-y-0.5 text-[13px] leading-5 text-foreground/80", clamp && "line-clamp-6")} />
          )}
          {!open && <EmbedFactLines facts={facts} />}
          {open && embed.fields && embed.fields.length > 0 && <EmbedFieldGrid fields={embed.fields} />}
          <BotMediaImage url={embed.image?.url} className="mt-1 max-h-72 w-full max-w-md rounded-md border border-border object-cover" />
        </div>
        <BotMediaImage url={embed.thumbnail?.url} className="size-16 shrink-0 self-start rounded-md border border-border object-cover" />
      </div>
      {(hasFooter || foldLabel) && (
        <div className="mt-1 flex min-h-5 flex-wrap items-center gap-x-3">
          {hasFooter && <EmbedFooter embed={embed} />}
          {foldLabel && <FoldBar label={open ? "Show less" : foldLabel} open={open} onToggle={() => setOpen(!open)} />}
        </div>
      )}
    </article>
  );
};
