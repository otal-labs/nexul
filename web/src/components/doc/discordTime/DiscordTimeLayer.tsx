import { useMemo, useState, type MouseEvent, type ReactNode } from "react";

import { splitDiscordTimes, type DiscordTime } from "@nexul/client-core/discordTime";

import { Popover, PopoverAnchor, PopoverContent } from "@/components/ui/popover";
import { DiscordTimeCard } from "@/components/chat/DiscordTimeCard";

interface OpenTime {
  time: DiscordTime;
  chip: HTMLElement;
}

const openTimeAt = (target: EventTarget): OpenTime | null => {
  const chip = target instanceof Element ? target.closest<HTMLElement>("[data-discord-time]") : null;
  const part = splitDiscordTimes(chip?.dataset.discordTime ?? "")[0];
  if (!chip || part?.kind !== "time") return null;
  return { time: part.time, chip };
};

// The editor's chips are plain DOM, so one popover per editor opens at whichever chip was clicked.
export const DiscordTimeLayer = ({ children }: { children: ReactNode }) => {
  const [open, setOpen] = useState<OpenTime | null>(null);
  const anchor = useMemo(() => ({ current: open?.chip ?? null }), [open]);
  const onClick = (event: MouseEvent) => {
    const found = openTimeAt(event.target);
    if (found) setOpen(found);
  };

  return (
    <div className="contents" onClick={onClick}>
      {children}
      <Popover open={open !== null} onOpenChange={(next) => !next && setOpen(null)}>
        {open && <PopoverAnchor virtualRef={anchor} />}
        <PopoverContent className="w-80">{open && <DiscordTimeCard time={open.time} />}</PopoverContent>
      </Popover>
    </div>
  );
};
