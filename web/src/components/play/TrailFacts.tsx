import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { microheaderClass } from "@/components/Microheader";
import { TrailFoundMemories } from "@/components/play/TrailFoundMemories";
import { TrailVersion } from "@/components/play/TrailVersion";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { computersQuery, useFetchHarnessProviders } from "@/hooks/PairingHooks";
import { useLiveTrailState } from "@/hooks/TrailHooks";
import { cn } from "@/lib/utils";
import { runnerPresence } from "@/models/ComputerChecks";
import { findModel } from "@/models/ModelPick";
import { passOverLines } from "@/models/Pairing";
import { sharedComputerName } from "@/models/SharedComputer";
import { isTrailActive, type Trail } from "@/models/Trail";
import { useIsViewingAs } from "@/stores/viewAsStore";

interface TrailFactsProps {
  trail: Trail;
}

interface FactProps {
  label: string;
  className?: string;
  children: ReactNode;
}

const Fact = ({ label, className, children }: FactProps) => (
  <div className={cn("min-w-0 space-y-1.5", className)}>
    <dt className={microheaderClass}>{label}</dt>
    <dd className="text-sm text-foreground">{children}</dd>
  </div>
);

const Muted = ({ children }: { children: ReactNode }) => <span className="text-muted-foreground">{children}</span>;

const NOT_IN_VIEW = "Not shown while viewing as another member";

// The computer by the name the viewer knows it by and, when a runner reaches it, whether that runner is connected now.
const ComputerName = ({ trail }: TrailFactsProps) => {
  const viewing = useIsViewingAs();
  const { data: computers, isError } = useQuery({ ...computersQuery, enabled: trail.computer_id !== "" && !viewing });
  const computer = computers?.find((c) => c.id === trail.computer_id);
  const presence = computer?.runner && runnerPresence(computer.runner);

  return (
    <>
      {trail.computer_id === "" && <Muted>None of your computers</Muted>}
      {trail.computer_id !== "" && viewing && <Muted>{NOT_IN_VIEW}</Muted>}
      {trail.computer_id !== "" && !viewing && isError && <Muted>Couldn't load the computer</Muted>}
      {trail.computer_id !== "" && !viewing && !isError && !computers && <Muted>Loading…</Muted>}
      {computers && !computer && <Muted>A computer you can't see</Muted>}
      {computer && <span className="font-medium break-words">{sharedComputerName(computer)}</span>}
      {presence && <SettingsStatus tone={presence.tone}>{presence.text}</SettingsStatus>}
    </>
  );
};

// The computer, then each one the run passed over before it and why.
const TrailComputer = ({ trail }: TrailFactsProps) => {
  const passedOver = trail.passed_over ?? [];
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <ComputerName trail={trail} />
      {passedOver.length > 0 && <span className="text-xs text-pretty text-muted-foreground">Passed over: {passOverLines(passedOver)}</span>}
    </span>
  );
};

// The model by the name its computer lists it under, its slug while the computer can't say, and the computer's default when none was picked.
const TrailModel = ({ trail }: TrailFactsProps) => {
  const viewing = useIsViewingAs();
  const { data: providers } = useFetchHarnessProviders(trail.model === "" || viewing ? "" : trail.computer_id);
  const provider = providers?.find((p) => p.id === trail.provider);
  const model = findModel(providers ?? [], { provider: trail.provider, model: trail.model });

  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      {trail.model === "" && <Muted>The computer's default</Muted>}
      {model && <span className="font-medium break-words">{model.name}</span>}
      {trail.model !== "" && !model && <span className="font-mono text-xs leading-5 break-all">{trail.model}</span>}
      {trail.model !== "" && <span className="text-xs text-muted-foreground">{provider?.name ?? trail.provider}</span>}
    </span>
  );
};

// The memories it was told to read, by their titles once the project's list answers; a deleted one keeps a placeholder, unlinked.
const PickedMemories = ({ trail }: TrailFactsProps) => {
  const { data: memories } = useFetchMemoriesByProject(trail.project_id);
  const picked = trail.selected_memory_ids.map((id) => {
    const memory = memories?.find((m) => m.id === id);
    return { id, title: memory?.title ?? "Deleted memory", deleted: !memory };
  });

  return (
    <>
      {picked.length === 0 && <Muted>None picked</Muted>}
      {picked.length > 0 && !memories && <Muted>Loading…</Muted>}
      {picked.length > 0 && memories && <TrailFoundMemories projectId={trail.project_id} memories={picked} />}
    </>
  );
};

// Where the run went and what it was given, in one row whose places hold for every run, then the memories under it.
export const TrailFacts = ({ trail }: TrailFactsProps) => {
  const state = useLiveTrailState(trail);

  return (
    <dl className="grid grid-cols-3 gap-x-8 gap-y-5">
      <Fact label={isTrailActive(state) ? "Runs on" : "Ran on"}>
        <TrailComputer trail={trail} />
      </Fact>
      <Fact label="Model">
        <TrailModel trail={trail} />
      </Fact>
      <Fact label="Instructions">
        {trail.instructions_version !== undefined && <TrailVersion trail={trail} version={trail.instructions_version} />}
        {trail.instructions_version === undefined && <Muted>Version not recorded</Muted>}
      </Fact>
      <Fact label="Memories" className="col-span-full">
        <PickedMemories trail={trail} />
      </Fact>
      {trail.found_memories && trail.found_memories.length > 0 && (
        <Fact label="Read by itself" className="col-span-full">
          <TrailFoundMemories projectId={trail.project_id} memories={trail.found_memories} />
        </Fact>
      )}
    </dl>
  );
};
