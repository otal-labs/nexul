import type { ReactNode } from "react";

import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { personLabel } from "@/models/Person";
import type { Trail } from "@/models/Trail";

interface TrailFactsProps {
  trail: Trail;
}

interface FactProps {
  label: string;
  children: ReactNode;
}

const Fact = ({ label, children }: FactProps) => (
  <div className="space-y-0.5">
    <dt className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">{label}</dt>
    <dd className="text-sm">{children}</dd>
  </div>
);

const formatTimestamp = (iso: string) => new Date(iso).toLocaleString();

// The run's facts as a two-column definition list: who, via what, when, and the choices made at the press.
export const TrailFacts = ({ trail }: TrailFactsProps) => {
  const lookup = usePersonLookup(trail.workspace_id);
  const { data: memories } = useFetchMemoriesByProject(trail.project_id);

  const memoryTitles = trail.selected_memory_ids.map((id) => memories?.find((m) => m.id === id)?.title ?? id);

  return (
    <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2">
      <Fact label="Play">{trail.play_label}</Fact>
      <Fact label="Started by">
        <span className="text-xs">{personLabel(lookup(trail.starter_id))}</span>
      </Fact>
      <Fact label="Via">
        <span className="font-mono text-xs">{trail.via}</span>
      </Fact>
      <Fact label="Started">
        <span className="font-mono text-xs tabular-nums">{formatTimestamp(trail.started_at)}</span>
      </Fact>
      <Fact label="Ended">
        <span className="font-mono text-xs tabular-nums">{trail.ended_at ? formatTimestamp(trail.ended_at) : "—"}</span>
      </Fact>
      <Fact label="Memories">
        {memoryTitles.length === 0 && <span className="text-muted-foreground">None</span>}
        {memoryTitles.length > 0 && memoryTitles.join(", ")}
      </Fact>
      <div className="sm:col-span-2">
        <Fact label="Instructions">
          {trail.custom_instructions === "" && <span className="text-muted-foreground">None</span>}
          {trail.custom_instructions !== "" && <span className="whitespace-pre-wrap">{trail.custom_instructions}</span>}
        </Fact>
      </div>
    </dl>
  );
};
