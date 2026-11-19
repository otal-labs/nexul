import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { Microheader } from "@/components/Microheader";
import { TrailFoundMemories } from "@/components/play/TrailFoundMemories";
import { TrailVersion } from "@/components/play/TrailVersion";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { computersQuery, useFetchHarnessProviders } from "@/hooks/PairingHooks";
import { useLiveTrailState } from "@/hooks/TrailHooks";
import { cn } from "@/lib/utils";
import { findModel } from "@/models/ModelPick";
import { passOverLines } from "@/models/Pairing";
import { sharedComputerName } from "@/models/SharedComputer";
import { isTrailActive, type Trail } from "@/models/Trail";
import { notViewing } from "@/stores/viewAsStore";

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
    <dt>
      <Microheader>{label}</Microheader>
    </dt>
    <dd className="text-sm text-foreground">{children}</dd>
  </div>
);

const None = ({ children }: { children: ReactNode }) => <span className="text-muted-foreground">{children}</span>;

// The computer by the name the viewer knows it by and whether its runner is connected now; a run that landed nowhere says so.
const TrailComputer = ({ trail }: TrailFactsProps) => {
  const passedOver = trail.passed_over ?? [];
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <ComputerName trail={trail} />
      {passedOver.length > 0 && <span className="text-xs text-pretty text-muted-foreground">Passed over: {passOverLines(passedOver)}</span>}
    </span>
  );
};

const ComputerName = ({ trail }: TrailFactsProps) => {
  const { data: computer, isPending } = useQuery({
    ...computersQuery,
    select: (computers) => computers.find((c) => c.id === trail.computer_id),
    enabled: trail.computer_id !== "" && notViewing(),
  });
  if (trail.computer_id === "") return <None>None of your computers</None>;
  if (isPending) return <None>Loading…</None>;
  if (!computer) return <None>A computer you can't see</None>;

  const connected = computer.runner?.connected === true;
  return (
    <>
      <span className="font-medium break-words">{sharedComputerName(computer)}</span>
      <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
        <span aria-hidden className={cn("size-2 shrink-0 rounded-full", connected ? "bg-success" : "border border-muted-foreground")} />
        {connected ? "Online" : "Offline"}
      </span>
    </>
  );
};

// The model by the name its computer lists it under, its slug while the computer can't say, and the computer's default when none was picked.
const TrailModel = ({ trail }: TrailFactsProps) => {
  const { data: providers } = useFetchHarnessProviders(trail.model === "" ? "" : trail.computer_id);
  if (trail.model === "") return <None>The computer's default</None>;

  const provider = providers?.find((p) => p.id === trail.provider);
  const model = findModel(providers ?? [], { provider: trail.provider, model: trail.model });
  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      {model && <span className="font-medium break-words">{model.name}</span>}
      {!model && <span className="font-mono text-xs break-all">{trail.model}</span>}
      <span className="text-xs text-muted-foreground">{provider?.name ?? trail.provider}</span>
    </span>
  );
};

// Where the run went and what it was given, in one row whose places hold for every run, then the memories under it.
export const TrailFacts = ({ trail }: TrailFactsProps) => {
  const state = useLiveTrailState(trail);
  const { data: memories } = useFetchMemoriesByProject(trail.project_id);
  const picked = trail.selected_memory_ids.map((id) => {
    const memory = memories?.find((m) => m.id === id);
    return { id, title: memory?.title ?? "Unknown memory", deleted: !memory };
  });

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
        {trail.instructions_version === undefined && <None>Not recorded</None>}
      </Fact>
      <Fact label="Memories" className="col-span-full">
        {picked.length === 0 && <None>None picked</None>}
        {picked.length > 0 && <TrailFoundMemories projectId={trail.project_id} memories={picked} />}
      </Fact>
      {trail.found_memories && trail.found_memories.length > 0 && (
        <Fact label="Read by itself" className="col-span-full">
          <TrailFoundMemories projectId={trail.project_id} memories={trail.found_memories} />
        </Fact>
      )}
    </dl>
  );
};
