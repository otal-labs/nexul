import { useState } from "react";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedFieldGrid } from "@/components/chat/EmbedFieldGrid";
import { EmbedFooter } from "@/components/chat/EmbedFooter";
import { EmbedHeading } from "@/components/chat/EmbedHeading";
import { FoldBar } from "@/components/chat/FoldBar";
import { cn } from "@/lib/utils";
import { EMBED_FIELD_LIMIT, embedFold, embedFoldLabel, httpUrl, type Embed } from "@/models/Embed";

// One embed behind a neutral rule: the sender's color would misread as a status hue, so it is never painted.
export const EmbedCard = ({ embed }: { embed: Embed }) => {
  const [open, setOpen] = useState(false);
  const fold = embedFold(embed);
  const foldLabel = embedFoldLabel(fold);
  const fields = open ? (embed.fields ?? []) : (embed.fields ?? []).slice(0, EMBED_FIELD_LIMIT);
  const image = httpUrl(embed.image?.url);
  const thumbnail = httpUrl(embed.thumbnail?.url);

  return (
    <div className="flex gap-3 border-l-2 border-muted-foreground/30 py-0.5 pl-3">
      <div className="min-w-0 flex-1 space-y-1.5">
        <EmbedHeading embed={embed} />
        {embed.description && (
          <DiscordMarkdown
            text={embed.description}
            className={cn("space-y-0.5 text-sm text-foreground/80", fold.longDescription && !open && "line-clamp-6")}
          />
        )}
        {fields.length > 0 && <EmbedFieldGrid fields={fields} />}
        {foldLabel && <FoldBar label={open ? "Show less" : foldLabel} open={open} onToggle={() => setOpen(!open)} />}
        {image && (
          <img src={image} alt="" loading="lazy" referrerPolicy="no-referrer" className="mt-1 max-h-72 w-full max-w-md rounded-md border border-border object-cover" />
        )}
        {(embed.footer || embed.timestamp) && <EmbedFooter embed={embed} />}
      </div>
      {thumbnail && (
        <img src={thumbnail} alt="" loading="lazy" referrerPolicy="no-referrer" className="size-16 shrink-0 rounded-md border border-border object-cover" />
      )}
    </div>
  );
};
