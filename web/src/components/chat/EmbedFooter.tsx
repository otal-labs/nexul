import { embedTimestamp, type Embed } from "@nexul/client-core/embed";

import { BotMediaImage } from "@/components/chat/BotMediaImage";
import { cn } from "@/lib/utils";
import { formatFullTime, formatVersionTime } from "@/utils/TimeUtility";

export const EmbedFooter = ({ embed: { footer, timestamp }, className }: { embed: Embed; className?: string }) => {
  const at = timestamp && embedTimestamp(timestamp);
  return (
    <p className={cn("flex min-w-0 items-center gap-1.5 font-mono text-[11px] text-muted-foreground", className)}>
      <BotMediaImage url={footer?.icon_url} className="size-4 shrink-0 rounded-full" />
      {footer && <span className="truncate">{footer.text}</span>}
      {footer && timestamp && <span aria-hidden>·</span>}
      {at && (
        <time dateTime={at} title={formatFullTime(at)} className="shrink-0">
          {formatVersionTime(at)}
        </time>
      )}
    </p>
  );
};
