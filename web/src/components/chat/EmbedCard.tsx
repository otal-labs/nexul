import { useState } from "react";

import { embedFacts, embedFactsFoldLabel, embedFold, embedReportTone, type Embed } from "@nexul/client-core/embed";

import { BotMediaImage } from "@/components/chat/BotMediaImage";
import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedFactLines } from "@/components/chat/EmbedFactLines";
import { EmbedFieldGrid } from "@/components/chat/EmbedFieldGrid";
import { EmbedFooter } from "@/components/chat/EmbedFooter";
import { EmbedHeading } from "@/components/chat/EmbedHeading";
import { toneBg } from "@/components/chat/EmbedTone";
import { FoldBar } from "@/components/chat/FoldBar";
import { cn } from "@/lib/utils";

const DESCRIPTION_LINES = 6;

// One embed as a report on the panel, its state on the leading edge; opened, its facts lay out as the sender's grid.
export const EmbedCard = ({ embed }: { embed: Embed }) => {
  const [open, setOpen] = useState(false);
  const facts = embedFacts(embed.fields);
  const foldLabel = embedFactsFoldLabel(embed, facts);
  const tone = embedReportTone(embed, facts);
  const hasFooter = !!(embed.footer || embed.timestamp);
  const clamp = embedFold(embed).longDescription && !open;
  // A many-line description keeps its first six lines whole, since a clamp cut the last or ended it in an ellipsis after a full stop.
  const manyLines = clamp && (embed.description ?? "").split("\n").length > DESCRIPTION_LINES;

  return (
    <article className="@container relative pl-4">
      <span aria-hidden className={cn("absolute inset-y-0.5 left-0 w-[3px] rounded-full", tone ? toneBg[tone] : "bg-muted-foreground/40")} />
      <div className="flex flex-col-reverse gap-3 @[22rem]:flex-row">
        <div className="min-w-0 flex-1 space-y-1">
          <EmbedHeading embed={embed} tone={tone} />
          {embed.description && (
            <DiscordMarkdown
              text={embed.description}
              maxLines={manyLines ? DESCRIPTION_LINES : undefined}
              className={cn("space-y-0.5 text-[13px] leading-5 text-foreground/80", clamp && !manyLines && "line-clamp-6")}
            />
          )}
          {!open && <EmbedFactLines facts={facts} />}
          {open && embed.fields && embed.fields.length > 0 && <EmbedFieldGrid fields={embed.fields} />}
          <BotMediaImage url={embed.image?.url} className="mt-1 max-h-72 w-full max-w-md rounded-md border border-border object-cover" />
        </div>
        <BotMediaImage url={embed.thumbnail?.url} className="size-16 shrink-0 self-start rounded-md border border-border object-cover" />
      </div>
      {(hasFooter || foldLabel) && (
        <div className="mt-1 flex min-h-5 flex-wrap items-baseline gap-x-3">
          {hasFooter && <EmbedFooter embed={embed} />}
          {foldLabel && <FoldBar label={open ? "Show less" : foldLabel} open={open} onToggle={() => setOpen(!open)} />}
        </div>
      )}
    </article>
  );
};
