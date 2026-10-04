import { CornerDownRight } from "lucide-react";

import { Textarea } from "@/components/ui/textarea";
import { ROUNDS } from "@/components/doc/prototype/ClarifyPrototypeData";
import { useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";

interface ClarifyPrototypeNoteProps {
  round: number;
}

// The round's closing "Anything else?" box while the round is the newest one.
export const ClarifyPrototypeNoteBox = ({ round }: ClarifyPrototypeNoteProps) => {
  const note = useClarifyPrototypeStore((s) => s.notes[round] ?? "");
  const setNote = useClarifyPrototypeStore((s) => s.setNote);
  const readOnly = useClarifyPrototypeStore((s) => s.phase === "closed");
  const id = `clarify-note-${round}`;
  return (
    <div className="px-1.5 pt-4">
      <label htmlFor={id} className="text-sm font-medium">
        Anything else?
      </label>
      <p className="text-xs text-muted-foreground">Optional. A question of your own, or something these didn't cover.</p>
      <Textarea
        id={id}
        value={note}
        readOnly={readOnly}
        onChange={(e) => setNote(round, e.target.value)}
        placeholder="Type it here"
        className="mt-2 min-h-14 text-sm"
      />
    </div>
  );
};

// A past round's "Anything else?" as asked, with the next round's one-line reply under it.
export const ClarifyPrototypeNoteReply = ({ round }: ClarifyPrototypeNoteProps) => {
  const note = useClarifyPrototypeStore((s) => s.notes[round] ?? "");
  const answered = useClarifyPrototypeStore((s) => round < s.rounds || s.phase === "nogaps" || s.phase === "closed");
  const reply = answered ? (ROUNDS[round - 1]?.reply ?? "") : "";
  if (note.trim() === "") return null;
  return (
    <div className="mt-2 space-y-1 px-1.5 text-sm">
      <p className="font-mono text-[11px] text-muted-foreground">Anything else?</p>
      <p className="text-muted-foreground">{note}</p>
      {reply !== "" && (
        <p className="flex items-start gap-1.5">
          <CornerDownRight className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span>{reply}</span>
        </p>
      )}
    </div>
  );
};
