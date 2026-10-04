import { SourcesProtoChecklist } from "@/components/memory/prototype/SourcesProtoChecklist";
import { SourcesProtoDraftButton, SourcesProtoNoDraftLine, SourcesProtoRunLine } from "@/components/memory/prototype/SourcesProtoDraft";
import { SourcesProtoMemory, SourcesProtoMemoryLine } from "@/components/memory/prototype/SourcesProtoMemory";
import { SourcesProtoSourceList } from "@/components/memory/prototype/SourcesProtoSourceList";
import { sourcesMeta, useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";

const Microheader = ({ label, meta }: { label: string; meta?: string | undefined }) => (
  <p className="flex items-baseline justify-between gap-3 px-1.5 font-mono text-[11px] text-muted-foreground uppercase">
    <span className="shrink-0">{label}</span>
    {meta && <span className="truncate normal-case tabular-nums">{meta}</span>}
  </p>
);

// The latest interview trail decides the header line: a drafting run while it works or until a memory is written.
const HeaderLine = () => {
  const run = useSourcesProtoStore((s) => s.run);
  const memory = useSourcesProtoStore((s) => s.memory);
  const drafting = run === "drafting" || (run === "ready" && !memory);
  return (
    <>
      {drafting && <SourcesProtoRunLine />}
      {!drafting && <SourcesProtoMemoryLine />}
    </>
  );
};

const Reads = () => {
  const sources = useSourcesProtoStore((s) => s.sources);
  return (
    <div className="space-y-2">
      <Microheader label="What the agent reads" meta={sources.length > 0 ? sourcesMeta(sources) : undefined} />
      <SourcesProtoSourceList />
      <div className="pt-4">
        <Microheader label="What it wrote" />
      </div>
    </div>
  );
};

export const SourcesProtoVariantB = () => (
  <div className="@container">
    <div className="grid gap-12 @5xl:grid-cols-[minmax(0,36rem)_minmax(0,1fr)] @5xl:items-start @5xl:gap-10">
      <SourcesProtoChecklist />
      <SourcesProtoMemory line={<HeaderLine />} actions={
          <>
            <SourcesProtoDraftButton />
            <SourcesProtoNoDraftLine className="basis-full" />
          </>
        }>
        <Reads />
      </SourcesProtoMemory>
    </div>
  </div>
);
