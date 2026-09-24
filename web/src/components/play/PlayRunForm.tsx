import { useState } from "react";

import { errorMessage } from "@/api/client";
import { EmptyRow } from "@/components/EmptyRow";
import { HarnessPickerPill, type HarnessPick } from "@/components/play/HarnessPickerPill";
import { MemoryPickRow } from "@/components/play/MemoryPickRow";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useRunPlay } from "@/hooks/TrailHooks";
import { useConfirmBlockedRun } from "@/hooks/useConfirmBlockedRun";
import { isWorkspaceMemory, type Memory } from "@/models/Memory";
import type { Play, PlayType } from "@/models/Play";
import type { BoardStatus } from "@/models/Status";
import type { LatestChoices } from "@/models/Trail";
import { cn } from "@/lib/utils";

interface PlayRunFormProps {
  play: Play;
  targetType: PlayType;
  targetId: string;
  memories: Memory[];
  columns: BoardStatus[];
  choices: LatestChoices;
  resolvedHarness: HarnessPick;
  canMoveTickets: boolean;
  onDone: () => void;
}

// Radix Select reserves "" for clearing, so "don't move" travels as this sentinel and leaves the request empty.
const NO_MOVE = "none";

const microheaderClass = "font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Mounted once per open, so the seed from the caller's latest trail needs no effect; a vanished column means no move.
export const PlayRunForm = ({
  play,
  targetType,
  targetId,
  memories,
  columns,
  choices,
  resolvedHarness,
  canMoveTickets,
  onDone,
}: PlayRunFormProps) => {
  const runPlay = useRunPlay();
  const confirmBlocked = useConfirmBlockedRun(targetType, targetId);
  const [selected, setSelected] = useState<string[]>(() =>
    choices.memory_ids.filter((id) => memories.some((m) => m.id === id && !m.always_included)),
  );
  const [instructions, setInstructions] = useState("");
  const [moveTo, setMoveTo] = useState(() =>
    canMoveTickets && columns.some((c) => c.id === choices.move_to_status_id) ? choices.move_to_status_id : NO_MOVE,
  );
  // Last choice for this user, play, and project wins; else the resolved target (spec.md, "the run dialog").
  const [harness, setHarness] = useState<HarnessPick>(() =>
    choices.computer_id !== ""
      ? { computer_id: choices.computer_id, provider: choices.provider, model: choices.model }
      : resolvedHarness,
  );
  const isTicket = play.type === "ticket";
  const column = isTicket && moveTo !== NO_MOVE && columns.find((c) => c.id === moveTo);
  const confirmLabel = column ? `Run ${play.label} · then ${column.name}` : `Run ${play.label}`;

  const workspaceMemories = memories.filter(isWorkspaceMemory);
  const projectMemories = memories.filter((memory) => !isWorkspaceMemory(memory));

  const toggle = (id: string) =>
    setSelected((current) => (current.includes(id) ? current.filter((m) => m !== id) : [...current, id]));

  const submit = async () => {
    if (!(await confirmBlocked(play.label))) return;
    runPlay.mutate(
      {
        playId: play.id,
        input: {
          target_type: targetType,
          target_id: targetId,
          memory_ids: selected,
          custom_instructions: instructions,
          move_to_status_id: column ? column.id : "",
          computer_id: harness.computer_id,
          provider: harness.provider,
          model: harness.model,
        },
      },
      { onSuccess: onDone },
    );
  };

  return (
    <>
      <section className="space-y-2">
        <h3 className={microheaderClass}>Memories</h3>
        {memories.length === 0 && <EmptyRow className="py-3">No memories in this project yet.</EmptyRow>}
        {memories.length > 0 && (
          <div className="rounded-md border border-border">
            {workspaceMemories.length > 0 && <p className={cn(microheaderClass, "px-3 pt-2")}>Workspace</p>}
            {workspaceMemories.map((memory) => (
              <MemoryPickRow
                key={memory.id}
                memory={memory}
                checked={memory.always_included || selected.includes(memory.id)}
                onToggle={() => toggle(memory.id)}
              />
            ))}
            {workspaceMemories.length > 0 && projectMemories.length > 0 && (
              <p className={cn(microheaderClass, "px-3 pt-2")}>This project</p>
            )}
            {projectMemories.map((memory) => (
              <MemoryPickRow
                key={memory.id}
                memory={memory}
                checked={memory.always_included || selected.includes(memory.id)}
                onToggle={() => toggle(memory.id)}
              />
            ))}
          </div>
        )}
      </section>

      <section className="space-y-2">
        <h3 className={microheaderClass}>Instructions for this run</h3>
        <Textarea
          rows={3}
          aria-label="Instructions for this run"
          placeholder="Optional. Steer the play; these win over the play's own instructions."
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
        />
      </section>

      {isTicket && (
        <section className="space-y-2">
          <h3 className={microheaderClass}>On success, move to</h3>
          <Select value={moveTo} onValueChange={setMoveTo} disabled={!canMoveTickets}>
            <SelectTrigger className="w-full" aria-label="On success, move to">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={NO_MOVE}>Don't move</SelectItem>
              {columns.map((c) => (
                <SelectItem key={c.id} value={c.id}>
                  {c.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {!canMoveTickets && (
            <p className="font-mono text-[11px] text-muted-foreground">You can't move tickets in this project.</p>
          )}
          {canMoveTickets && (
            <p className="font-mono text-[11px] text-muted-foreground">
              Never moves backwards; skipped with a note if the ticket has already moved on.
            </p>
          )}
        </section>
      )}

      {runPlay.error && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage(runPlay.error)}
        </p>
      )}

      <div className="flex justify-start">
        <HarnessPickerPill value={harness} onChange={setHarness} />
      </div>

      <DialogFooter className="gap-2 sm:gap-0">
        <Button variant="ghost" onClick={onDone}>
          Cancel
        </Button>
        <Button onClick={() => void submit()} disabled={runPlay.isPending}>
          {confirmLabel}
        </Button>
      </DialogFooter>
    </>
  );
};
