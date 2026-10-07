import { embedTimestamp, httpUrl, type Embed } from "@/models/Embed";
import { formatCalendarTime, formatFullTime } from "@/utils/TimeUtility";

export const EmbedFooter = ({ embed: { footer, timestamp } }: { embed: Embed }) => {
  const icon = httpUrl(footer?.icon_url);
  const at = timestamp && embedTimestamp(timestamp);
  return (
    <p className="flex min-w-0 items-center gap-1.5 pt-0.5 text-[11px] leading-4 font-medium text-muted-foreground">
      {icon && <img src={icon} alt="" loading="lazy" referrerPolicy="no-referrer" className="size-4 shrink-0 rounded-full" />}
      {footer && <span className="truncate">{footer.text}</span>}
      {footer && timestamp && <span aria-hidden>•</span>}
      {at && (
        <time dateTime={at} title={formatFullTime(at)} className="shrink-0">
          {formatCalendarTime(at)}
        </time>
      )}
    </p>
  );
};
