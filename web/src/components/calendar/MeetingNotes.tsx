import { useState } from "react";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { Microheader } from "@/components/Microheader";
import { markClamped } from "@/lib/markClamped";
import { cn } from "@/lib/utils";

// A meeting's notes under the people, held to eight lines until opened, so who's coming stays in view.
export const MeetingNotes = ({ notes }: { notes: string }) => {
  const [open, setOpen] = useState(false);
  return (
    <section aria-label="Notes">
      <Microheader>Notes</Microheader>
      <div ref={markClamped} className={cn("peer overflow-hidden", !open && "line-clamp-8")}>
        <DiscordMarkdown text={notes} className="space-y-0.5 text-sm" />
      </div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={cn(
          "mt-1 hidden text-xs text-muted-foreground transition-colors duration-150 ease-standard peer-data-clamped:inline-flex hover:text-foreground",
          open && "inline-flex",
        )}
      >
        {open ? "Show less" : "Show all"}
      </button>
    </section>
  );
};
