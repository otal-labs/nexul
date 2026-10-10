import { useCallback, useState } from "react";

import { microheaderClass } from "@/components/Microheader";
import { HarnessPickerPill, type HarnessPick } from "@/components/play/HarnessPickerPill";
import { MemoryPickSection } from "@/components/play/MemoryPickSection";
import { PlayRunError } from "@/components/play/PlayRunError";
import { RunWhereSection, type RunWhere } from "@/components/play/RunWhereSection";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { useRunPlay } from "@/hooks/TrailHooks";
import { useConfirmBlockedRun } from "@/hooks/useConfirmBlockedRun";
import type { Memory } from "@/models/Memory";
import { isNeedsLocationRefusal } from "@/models/Pairing";
import type { Play, PlayType } from "@/models/Play";
import type { LatestChoices } from "@/models/Trail";

interface PlayRunFormProps {
  play: Play;
  targetType: PlayType;
  targetId: string;
  memories: Memory[];
  choices: LatestChoices;
  resolvedHarness: HarnessPick;
  // The person's project link, else their defaults as the suggestion; linked says which.
  where: RunWhere;
  linked: boolean;
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
  where: seedWhere,
  linked,
  onDone,
}: PlayRunFormProps) => {
  const runPlay = useRunPlay();
  const confirmBlocked = useConfirmBlockedRun(targetType, targetId);
  const [selected, setSelected] = useState<string[]>(() =>
    choices.memory_ids.filter((id) => memories.some((m) => m.id === id && !m.always_included)),
  );
  const [instructions, setInstructions] = useState("");
  const [where, setWhere] = useState<RunWhere>(seedWhere);
  const [changing, setChanging] = useState(false);
  // The model last picked on this computer for this play wins; else the resolved one.
  const [harness, setHarness] = useState<HarnessPick>(() =>
    choices.computer_id === seedWhere.computer_id && choices.computer_id !== ""
      ? { computer_id: choices.computer_id, provider: choices.provider, model: choices.model, model_options: choices.model_options ?? [] }
      : { ...resolvedHarness, computer_id: seedWhere.computer_id },
  );
  const asking = !linked || changing || isNeedsLocationRefusal(runPlay.error);
  const whereMissing = asking && (where.computer_id === "" || where.harness_project_id === "");

  // Another computer has its own providers, so the model goes back to that computer's defaults.
  const pickWhere = useCallback((next: RunWhere) => {
    setWhere(next);
    setHarness((current) =>
      current.computer_id === next.computer_id ? current : { computer_id: next.computer_id, provider: "", model: "", model_options: [] },
    );
  }, []);
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
          computer_id: where.computer_id,
          ...(asking && { harness_project_id: where.harness_project_id }),
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
          placeholder="Optional. These override the play's own instructions."
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

      <RunWhereSection
        value={where}
        asking={asking}
        firstRun={!linked}
        onChange={pickWhere}
        onChangeRequested={() => setChanging(true)}
      />

      {runPlay.error && <PlayRunError error={runPlay.error} />}

      <div className="flex justify-start">
        <HarnessPickerPill value={harness} onChange={setHarness} />
      </div>

      <DialogFooter className="gap-2 sm:gap-0">
        <Button variant="ghost" onClick={onDone}>
          Cancel
        </Button>
        <Button onClick={() => void submit()} loading={runPlay.isPending} disabled={whereMissing}>
          Run {play.label}
        </Button>
      </DialogFooter>
    </>
  );
};
