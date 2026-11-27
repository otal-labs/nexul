import { useState } from "react";
import { XIcon } from "lucide-react";

import { LabelChip } from "@/components/LabelChip";
import { Input } from "@/components/ui/input";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import type { Computer } from "@/models/Pairing";

interface ComputerLabelsProps {
  computer: Computer;
}

// The labels a play's needs match (ADR 0197): the ones you set, removable, then on their own line those its facts give it. Sits in a Labels row.
export const ComputerLabels = ({ computer }: ComputerLabelsProps) => {
  const update = useUpdateComputerSettings(computer.id);
  const [draft, setDraft] = useState("");
  const own = computer.labels ?? [];
  const derived = computer.derived_labels ?? [];

  const add = () => {
    const label = draft.trim().toLowerCase();
    if (label === "" || own.includes(label)) return;
    update.mutate({ labels: [...own, label] }, { onSuccess: () => setDraft("") });
  };

  return (
    <div className="space-y-1.5 text-xs">
      <div className="flex flex-wrap items-center gap-1.5">
        {own.map((label) => (
          <LabelChip key={label} label={label} className="pr-1">
            <button
              type="button"
              aria-label={`Remove ${label}`}
              disabled={update.isPending}
              onClick={() => update.mutate({ labels: own.filter((l) => l !== label) })}
              className="-my-0.5 ml-0.5 self-center rounded-full p-0.5 text-muted-foreground hover:text-foreground"
            >
              <XIcon aria-hidden className="size-3" />
            </button>
          </LabelChip>
        ))}
        <Input
          aria-label={`Add a label to ${computer.name}`}
          placeholder="Add label"
          value={draft}
          disabled={update.isPending}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key !== "Enter") return;
            e.preventDefault();
            add();
          }}
          className="h-6 w-28 px-2 text-xs"
        />
      </div>
      {derived.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-muted-foreground">From its facts</span>
          {derived.map((label) => (
            <LabelChip key={label} label={label} className="border-dashed" />
          ))}
        </div>
      )}
    </div>
  );
};
