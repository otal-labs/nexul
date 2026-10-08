import { CornerDownRight } from "lucide-react";

import type { ClarificationRound } from "@/models/Clarification";

interface AnythingElseReplyProps {
  round: ClarificationRound;
}

// A round's "Anything else?" as written, with the next round's one-line reply under it; shown even while the round is folded.
export const AnythingElseReply = ({ round }: AnythingElseReplyProps) => {
  if (round.anything_else === "") return null;
  return (
    <div className="mt-2 space-y-1 px-1.5 text-sm">
      <p className="font-mono text-xs text-muted-foreground">Anything else?</p>
      <p className="whitespace-pre-line text-muted-foreground">{round.anything_else}</p>
      {round.anything_else_reply !== "" && (
        <p className="flex items-start gap-1.5">
          <CornerDownRight className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span>{round.anything_else_reply}</span>
        </p>
      )}
    </div>
  );
};
