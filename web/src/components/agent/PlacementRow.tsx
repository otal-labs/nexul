import { LabelChip } from "@/components/LabelChip";
import { passOverLines, type PlacementAnswer } from "@/models/Pairing";
import type { WorkspacePlay } from "@/models/Play";

interface PlacementRowProps {
  answer: PlacementAnswer;
  plays: WorkspacePlay[];
  nameOf: (computerId: string) => string;
}

// One set of needs: the plays naming it, the computer that would take their run now, and who it passed over first.
export const PlacementRow = ({ answer, plays, nameOf }: PlacementRowProps) => (
  <li className="space-y-1 px-3 py-2.5 text-sm">
    <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
      <span className="min-w-0 flex-1 break-words">{plays.map((p) => p.label).join(", ")}</span>
      {answer.needs.map((need) => (
        <LabelChip key={need} label={need} />
      ))}
      {answer.computer_id && <span className="font-medium">{nameOf(answer.computer_id)}</span>}
      {!answer.computer_id && <span className="text-muted-foreground">No computer can take it</span>}
    </div>
    {answer.passed_over.length > 0 && <p className="text-xs break-words text-muted-foreground">Passed over: {passOverLines(answer.passed_over)}</p>}
  </li>
);
