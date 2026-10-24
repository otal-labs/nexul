import { CalendarPlus } from "lucide-react";
import { useContext } from "react";

import { discordTimeIcs, fullDiscordTime, type DiscordTime } from "@nexul/client-core/discordTime";

import { Button } from "@/components/ui/button";
import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { CopyButton } from "@/components/settings/CopyButton";
import { saveTextFile } from "@/utils/ContainerLogUtility";

const icsFilename = (title: string): string => `${title.replace(/[^\p{L}\p{N}]+/gu, "-").replace(/^-|-$/g, "").toLowerCase() || "event"}.ics`;

// What a timestamp's popover holds: the full time, the code as written, and the two ways to take it elsewhere.
export const DiscordTimeCard = ({ time }: { time: DiscordTime }) => {
  const title = useContext(CalendarTitleContext);
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <p className="text-sm font-medium">{fullDiscordTime(time.date)}</p>
        <p className="font-mono text-xs break-all text-muted-foreground">{time.code}</p>
      </div>
      <div className="flex flex-wrap gap-2">
        <CopyButton value={time.code} label="Copy code" />
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => saveTextFile(icsFilename(title), discordTimeIcs(time.date, title), "text/calendar")}
        >
          <CalendarPlus className="size-3.5" aria-hidden />
          Add to calendar
        </Button>
      </div>
    </div>
  );
};
