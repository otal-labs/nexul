import { useState } from "react";

import { Dialog, DialogContent } from "@/components/ui/dialog";

import { DialogPill } from "@/components/DialogPill";
import { HandoffConversation } from "@/components/handoff/HandoffConversation";
import { HandoffStateDot } from "@/components/handoff/HandoffStateDot";
import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { HANDOFF_STATE_LABELS, type Handoff } from "@/models/Handoff";

interface HandoffPillProps {
  handoff: Handoff;
}

// One handed-off agent under the Agent's reply, opening its conversation in a dialog the width of the note's.
export const HandoffPill = ({ handoff }: HandoffPillProps) => {
  const [open, setOpen] = useState(false);
  return (
    <>
      <DialogPill
        icon={<HarnessProviderMark driver={handoff.driver} className="size-3" />}
        label={handoff.title}
        name={`${[handoff.title, handoff.model].filter(Boolean).join(" ")}, ${HANDOFF_STATE_LABELS[handoff.state]}`}
        trailing={
          <>
            {handoff.model && <span className="shrink-0 font-mono text-[11px] text-muted-foreground">{handoff.model}</span>}
            <HandoffStateDot state={handoff.state} />
          </>
        }
        onOpen={() => setOpen(true)}
      />
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[calc(100dvh-4rem)] overflow-y-auto bg-popover sm:max-w-[min(48rem,calc(100%-2rem))]">
          <HandoffConversation handoff={handoff} />
        </DialogContent>
      </Dialog>
    </>
  );
};
