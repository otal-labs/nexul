import { Plus } from "lucide-react";

import { SourcesProtoAddForm } from "@/components/memory/prototype/SourcesProtoAddForm";
import { SourcesProtoChecklist } from "@/components/memory/prototype/SourcesProtoChecklist";
import { SourcesProtoDraftButton, SourcesProtoNoDraftLine, SourcesProtoRunLine } from "@/components/memory/prototype/SourcesProtoDraft";
import { SourcesProtoMemory } from "@/components/memory/prototype/SourcesProtoMemory";
import { SourcesProtoSourceChip } from "@/components/memory/prototype/SourcesProtoSourceChip";
import { EMPTY_SOURCES } from "@/components/memory/prototype/SourcesProtoSourceList";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";

// One row across both columns: a chip per source, "+ Add source", and the drafting line with "Draft answers" at its end.
const Strip = () => {
  const sources = useSourcesProtoStore((s) => s.sources);
  const adding = useSourcesProtoStore((s) => s.adding);
  const setAdding = useSourcesProtoStore((s) => s.setAdding);
  return (
    <section aria-label="Sources" className="space-y-3 border-b border-border pb-4">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <h3 className="font-mono text-[11px] text-muted-foreground uppercase">Sources</h3>
        {sources.length === 0 && !adding && <p className="min-w-0 flex-1 basis-64 text-sm text-muted-foreground">{EMPTY_SOURCES}</p>}
        <div className="ml-auto flex flex-wrap items-center justify-end gap-3">
          <SourcesProtoRunLine />
          <SourcesProtoNoDraftLine />
          <SourcesProtoDraftButton />
        </div>
      </div>
      <ul className="flex flex-wrap gap-2 empty:hidden">
        {sources.map((source) => (
          <SourcesProtoSourceChip key={source.id} source={source} />
        ))}
        {!adding && (
          <li>
            <button
              type="button"
              onClick={() => setAdding("path")}
              className="flex h-9 items-center gap-1.5 rounded-md border border-dashed border-border px-3 text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40 hover:text-foreground"
            >
              <Plus className="size-3.5" aria-hidden />
              Add source
            </button>
          </li>
        )}
      </ul>
      {adding && (
        <div className="max-w-xl">
          <SourcesProtoAddForm />
        </div>
      )}
    </section>
  );
};

export const SourcesProtoVariantC = () => (
  <div className="@container space-y-8">
    <Strip />
    <div className="grid gap-12 @5xl:grid-cols-[minmax(0,36rem)_minmax(0,1fr)] @5xl:items-start @5xl:gap-10">
      <SourcesProtoChecklist />
      <SourcesProtoMemory />
    </div>
  </div>
);
