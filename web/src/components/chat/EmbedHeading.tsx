import { httpUrl, type Embed, type EmbedTone } from "@nexul/client-core/embed";

import { BotMediaImage } from "@/components/chat/BotMediaImage";
import { toneIcon, toneText } from "@/components/chat/EmbedTone";
import { cn } from "@/lib/utils";

const EmbedAuthor = ({ author }: { author: NonNullable<Embed["author"]> }) => {
  const url = httpUrl(author.url);
  return (
    <p className="flex min-w-0 items-center gap-1.5 text-xs font-medium text-muted-foreground">
      <BotMediaImage url={author.icon_url} className="size-4 shrink-0 rounded-full" />
      {url && (
        <a href={url} target="_blank" rel="noopener noreferrer" className="truncate hover:text-foreground hover:underline">
          {author.name}
        </a>
      )}
      {!url && <span className="truncate">{author.name}</span>}
    </p>
  );
};

const titleClass = "text-sm font-semibold leading-5 wrap-anywhere text-foreground";

// The author line over the title, which leads with the post's state and links out when the sender gave it a URL.
export const EmbedHeading = ({ embed: { author, title, url }, tone }: { embed: Embed; tone: EmbedTone | null }) => {
  const href = httpUrl(url);
  const Icon = tone && toneIcon[tone];
  return (
    <div className="space-y-1">
      {author && <EmbedAuthor author={author} />}
      {title && (
        <div className="flex items-start gap-2">
          {Icon && tone && <Icon aria-hidden className={cn("mt-0.5 size-4 shrink-0", toneText[tone])} />}
          {href && (
            <a href={href} target="_blank" rel="noopener noreferrer" className={cn(titleClass, "hover:underline")}>
              {title}
            </a>
          )}
          {!href && <p className={titleClass}>{title}</p>}
        </div>
      )}
    </div>
  );
};
