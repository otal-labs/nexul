import { embedTimestamp, type Embed } from "@nexul/client-core/embed";

import { BotMediaImage } from "@/components/chat/BotMediaImage";
import { formatCalendarTime, formatFullTime } from "@/utils/TimeUtility";

export const EmbedFooter = ({ embed: { footer, timestamp } }: { embed: Embed }) => {
  const at = timestamp && embedTimestamp(timestamp);
  return (
    <p className="flex min-w-0 items-center gap-1.5 font-mono text-[11px] text-muted-foreground">
      <BotMediaImage url={footer?.icon_url} className="size-4 shrink-0 rounded-full" />
      {footer && <span className="truncate">{footer.text}</span>}
      {footer && timestamp && <span aria-hidden>·</span>}
      {at && (
        <time dateTime={at} title={formatFullTime(at)} className="shrink-0">
          {formatCalendarTime(at)}
        </time>
      )}
    </p>
  );
};
