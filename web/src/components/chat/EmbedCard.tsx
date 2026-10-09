import { useState } from "react";

import { EMBED_FIELD_LIMIT, embedCardTone, embedFold, embedFoldLabel, type Embed } from "@nexul/client-core/embed";

import { BotMediaImage } from "@/components/chat/BotMediaImage";
import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedFieldGrid } from "@/components/chat/EmbedFieldGrid";
import { EmbedFooter } from "@/components/chat/EmbedFooter";
import { EmbedHeading } from "@/components/chat/EmbedHeading";
import { toneBg } from "@/components/chat/EmbedTone";
import { FoldBar } from "@/components/chat/FoldBar";
import { cn } from "@/lib/utils";

// One embed as a card with its state on the leading edge; a narrow pane stacks the thumbnail on top.
export const EmbedCard = ({ embed }: { embed: Embed }) => {
  const [open, setOpen] = useState(false);
  const fold = embedFold(embed);
  const foldLabel = embedFoldLabel(fold);
  const tone = embedCardTone(embed);
  const fields = open ? (embed.fields ?? []) : (embed.fields ?? []).slice(0, EMBED_FIELD_LIMIT);
  const hasFooter = !!(embed.footer || embed.timestamp);

  return (
    <article className="@container relative overflow-hidden rounded-lg bg-card shadow-card ring-1 ring-border">
      {tone && <span aria-hidden className={cn("absolute inset-y-0 left-0 w-[3px]", toneBg[tone])} />}
      <div className="flex flex-col-reverse gap-3 px-4 py-3 @[22rem]:flex-row">
        <div className="min-w-0 flex-1 space-y-2.5">
          <EmbedHeading embed={embed} tone={tone} />
          {embed.description && (
            <DiscordMarkdown
              text={embed.description}
              className={cn("space-y-0.5 text-[13px] leading-5 text-foreground/80", fold.longDescription && !open && "line-clamp-6")}
            />
          )}
          {fields.length > 0 && <EmbedFieldGrid fields={fields} />}
          <BotMediaImage url={embed.image?.url} className="mt-1 max-h-72 w-full max-w-md rounded-md border border-border object-cover" />
        </div>
        <BotMediaImage url={embed.thumbnail?.url} className="size-16 shrink-0 self-start rounded-md border border-border object-cover" />
      </div>
      {(hasFooter || foldLabel) && (
        <div className="flex min-h-9 items-center justify-between gap-3 border-t border-border px-4 py-1.5">
          {hasFooter && <EmbedFooter embed={embed} />}
          {foldLabel && <FoldBar className="ml-auto" label={open ? "Show less" : foldLabel} open={open} onToggle={() => setOpen(!open)} />}
        </div>
      )}
    </article>
  );
};
