import { ArrowRightIcon } from "lucide-react";

import { passOverLines, type PlacementAnswer } from "@/models/Pairing";
import type { WorkspacePlay } from "@/models/Play";

interface PlacementRowProps {
  play: WorkspacePlay;
  answer: PlacementAnswer;
  nameOf: (computerId: string) => string;
}

// One play and the computer that would take its run now, with who it passed over first.
export const PlacementRow = ({ play, answer, nameOf }: PlacementRowProps) => (
  <li className="space-y-0.5 px-3 py-2 text-sm">
    <div className="flex min-w-0 items-baseline gap-2">
      <span className="min-w-0 truncate">{play.label}</span>
      <ArrowRightIcon aria-hidden className="size-3 shrink-0 translate-y-0.5 text-muted-foreground" />
      {answer.computer_id && <span className="shrink-0 font-mono font-medium">{nameOf(answer.computer_id)}</span>}
      {!answer.computer_id && <span className="shrink-0 text-muted-foreground">No computer can take it</span>}
    </div>
    {answer.passed_over.length > 0 && <p className="text-xs break-words text-muted-foreground">Passed over: {passOverLines(answer.passed_over)}</p>}
  </li>
);
