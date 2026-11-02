import { useState } from "react";

import { Dialog, DialogContent } from "@/components/ui/dialog";

import { HandoffConversation } from "@/components/handoff/HandoffConversation";
import { HandoffStateDot } from "@/components/handoff/HandoffStateDot";
import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { HANDOFF_STATE_LABELS, type Handoff } from "@/models/Handoff";

interface HandoffRowProps {
  handoff: Handoff;
}

// One handed-off agent in the reply's sub-agents fold, opening its conversation in a dialog the width of the note's.
export const HandoffRow = ({ handoff }: HandoffRowProps) => {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-haspopup="dialog"
        aria-label={`${[handoff.title, handoff.model].filter(Boolean).join(" ")}, ${HANDOFF_STATE_LABELS[handoff.state]}`}
        className="flex w-full min-w-0 items-center gap-2 border-t border-border px-2.5 py-1.5 text-left text-[13px] leading-5 transition-colors duration-150 ease-standard hover:bg-accent"
      >
        <HarnessProviderMark driver={handoff.driver} className="size-3.5 shrink-0" />
        <span className="min-w-0 shrink truncate text-foreground">{handoff.title}</span>
        {handoff.model && <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">{handoff.model}</span>}
        {!handoff.model && <span className="flex-1" />}
        <HandoffStateDot state={handoff.state} />
      </button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[calc(100dvh-4rem)] overflow-y-auto bg-popover sm:max-w-[min(48rem,calc(100%-2rem))]">
          <HandoffConversation handoff={handoff} />
        </DialogContent>
      </Dialog>
    </>
  );
};
