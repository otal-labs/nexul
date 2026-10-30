import { useState } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { Microheader } from "@/components/Microheader";
import { useEntityActions } from "@/hooks/useEntityActions";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import type { Play } from "@/models/Play";
import { markClamped } from "@/lib/markClamped";
import { cn } from "@/lib/utils";

interface PlayInstructionsSectionProps {
  play: Play;
}

// Backticked spans read as code, the way the instructions' tool and branch names are written.
const InstructionText = ({ text }: { text: string }) =>
  text.split(/(`[^`\n]+`)/).map((part, i) => {
    const code = part.length > 2 && part.startsWith("`") && part.endsWith("`");
    return (
      <span key={i} className={cn(code && "rounded-sm bg-foreground/[0.06] px-1 py-0.5 font-mono text-[12px] leading-none box-decoration-clone")}>
        {code ? part.slice(1, -1) : part}
      </span>
    );
  });

export const PlayInstructionsSection = ({ play }: PlayInstructionsSectionProps) => {
  const [open, setOpen] = useState(false);
  const answer = useEntityActions("play", play.id);
  const { edit } = usePlayDialogs();

  return (
    <section aria-labelledby="play-instructions" className="space-y-3">
      <Microheader id="play-instructions" className="flex h-7 items-center">
        Instructions
      </Microheader>
      {play.instructions !== "" && (
        <div className="rounded-lg bg-surface-2 px-4 py-3">
          {/* Ten lines, faded, until opened, so the runs above stay the page's first read; Show all appears only when it cuts. */}
          <p
            ref={markClamped}
            className={cn(
              "peer max-w-[72ch] text-[13px] leading-relaxed whitespace-pre-wrap text-foreground [overflow-wrap:anywhere]",
              !open && "line-clamp-10 data-clamped:[mask-image:linear-gradient(to_bottom,black_70%,transparent)]",
            )}
          >
            <InstructionText text={play.instructions} />
          </p>
          <Button
            variant="ghost"
            size="sm"
            aria-expanded={open}
            className={cn("mt-1 -ml-2.5 hidden peer-data-clamped:inline-flex", open && "inline-flex")}
            onClick={() => setOpen(!open)}
          >
            {open ? "Show less" : "Show all"}
          </Button>
        </div>
      )}
      {play.instructions === "" && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border bg-card px-4 py-5 text-sm text-muted-foreground">
          No instructions yet
          <ActionButton permission="plays:write" answer={answer("plays:write")} verb="edit plays" variant="link" size="sm" className="h-auto p-0 text-sm" onClick={() => void edit(play)}>
            Write instructions
          </ActionButton>
        </div>
      )}
    </section>
  );
};
