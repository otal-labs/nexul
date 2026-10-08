import { useState } from "react";

import { microheaderClass } from "@/components/Microheader";
import { HarnessPickerPill, type HarnessPick } from "@/components/play/HarnessPickerPill";
import { MemoryPickSection } from "@/components/play/MemoryPickSection";
import { PlayRunError } from "@/components/play/PlayRunError";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { useRunPlay } from "@/hooks/TrailHooks";
import { useConfirmBlockedRun } from "@/hooks/useConfirmBlockedRun";
import type { Memory } from "@/models/Memory";
import type { Play, PlayType } from "@/models/Play";
import type { LatestChoices } from "@/models/Trail";

interface PlayRunFormProps {
  play: Play;
  targetType: PlayType;
  targetId: string;
  memories: Memory[];
  choices: LatestChoices;
  resolvedHarness: HarnessPick;
  onDone: () => void;
}

// Mounted once per open, so the seed from the caller's latest trail needs no effect.
export const PlayRunForm = ({
  play,
  targetType,
  targetId,
  memories,
  choices,
  resolvedHarness,
  onDone,
}: PlayRunFormProps) => {
  const runPlay = useRunPlay();
  const confirmBlocked = useConfirmBlockedRun(targetType, targetId);
  const [selected, setSelected] = useState<string[]>(() =>
    choices.memory_ids.filter((id) => memories.some((m) => m.id === id && !m.always_included)),
  );
  const [instructions, setInstructions] = useState("");
  // Last choice for this user, play, and project wins; else the resolved target (spec.md, "the run dialog").
  const [harness, setHarness] = useState<HarnessPick>(() =>
    choices.computer_id !== ""
      ? { computer_id: choices.computer_id, provider: choices.provider, model: choices.model, model_options: choices.model_options ?? [] }
      : resolvedHarness,
  );
  const regular = memories.filter((m) => !m.footer);
  const footers = memories.filter((m) => m.footer);

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
          computer_id: harness.computer_id,
          provider: harness.provider,
          model: harness.model,
          model_options: harness.model_options,
        },
      },
      { onSuccess: onDone },
    );
  };

  return (
    <>
      <MemoryPickSection
        title="Main"
        emptyMessage="No memories in this project yet."
        memories={regular}
        selected={selected}
        onToggle={toggle}
      />

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

      <MemoryPickSection
        title="Footer"
        emptyMessage="No footer memories. Move a memory to the Footer folder to conclude runs with it."
        memories={footers}
        selected={selected}
        onToggle={toggle}
      />

      {runPlay.error && <PlayRunError error={runPlay.error} />}

      <div className="flex justify-start">
        <HarnessPickerPill value={harness} onChange={setHarness} />
      </div>

      <DialogFooter className="gap-2 sm:gap-0">
        <Button variant="ghost" onClick={onDone}>
          Cancel
        </Button>
        <Button onClick={() => void submit()} loading={runPlay.isPending}>
          Run {play.label}
        </Button>
      </DialogFooter>
    </>
  );
};
