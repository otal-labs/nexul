import { CalendarClock } from "lucide-react";

import { ScheduledMessageRow } from "@/components/chat/ScheduledMessageRow";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchScheduledMessages } from "@/hooks/ScheduledMessageHooks";

// A line above the composer while the viewer has messages waiting to post here; it opens them to change or cancel.
export const ScheduledMessagesBar = ({ conversationId }: { conversationId: string }) => {
  const { data: scheduled } = useFetchScheduledMessages(conversationId);
  const count = scheduled?.length ?? 0;
  return (
    count > 0 && (
      <div className="mx-auto w-full max-w-[var(--chat-column,48rem)] px-3">
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="ghost" size="sm" className="h-7 gap-1.5 px-2 text-xs text-muted-foreground hover:text-foreground">
              <CalendarClock className="size-3.5" aria-hidden />
              {count === 1 ? "1 scheduled message" : `${count} scheduled messages`}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-[min(26rem,calc(100vw-2rem))] p-0">
            <ul className="max-h-80 divide-y divide-border overflow-y-auto">
              {scheduled?.map((s) => <ScheduledMessageRow key={s.id} scheduled={s} />)}
            </ul>
          </PopoverContent>
        </Popover>
      </div>
    )
  );
};
