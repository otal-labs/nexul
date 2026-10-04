import { Textarea } from "@/components/ui/textarea";
import { useSaveAnythingElse } from "@/hooks/DocHooks";
import type { ClarificationRound } from "@/models/Clarification";

interface AnythingElseBoxProps {
  round: ClarificationRound;
}

// The open round's closing "Anything else?" box, saved when it loses focus.
export const AnythingElseBox = ({ round }: AnythingElseBoxProps) => {
  const save = useSaveAnythingElse(round.doc_id);
  const id = `anything-else-${round.round}`;
  return (
    <div className="px-1.5 pt-4">
      <label htmlFor={id} className="text-sm font-medium">
        Anything else?
      </label>
      <p className="text-xs text-muted-foreground">Optional. A question of your own, or something these didn't cover.</p>
      <Textarea
        // Remounts on someone else's saved text, so the box shows it.
        key={round.anything_else}
        id={id}
        defaultValue={round.anything_else}
        disabled={save.isPending}
        onBlur={(e) => {
          if (e.target.value.trim() === round.anything_else) return;
          save.mutate({ round: round.round, text: e.target.value });
        }}
        placeholder="Type it here"
        className="mt-2 min-h-14 text-sm"
      />
    </div>
  );
};
