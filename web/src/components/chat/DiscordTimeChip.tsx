import { Clock } from "lucide-react";

import { formatDiscordTime, fullDiscordTime, type DiscordTime } from "@nexul/client-core/discordTime";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { DiscordTimeCard } from "@/components/chat/DiscordTimeCard";
import { useNow } from "@/hooks/useNow";

// data-url makes a copied selection carry the code, so a re-paste stays local for every reader (MessageBody's onCopy).
export const DiscordTimeChip = ({ time }: { time: DiscordTime }) => {
  const now = useNow(time.style === "R");
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button type="button" className="mention-chip" title={fullDiscordTime(time.date)} data-url={time.code}>
          <Clock className="size-[1em] shrink-0" aria-hidden />
          <time className="mention-chip__label" dateTime={time.date.toISOString()}>
            {formatDiscordTime(time, now)}
          </time>
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-80">
        <DiscordTimeCard time={time} />
      </PopoverContent>
    </Popover>
  );
};
