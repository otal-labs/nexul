import { ChevronDownIcon } from "lucide-react";

import { SourcesProtoChecklist } from "@/components/memory/prototype/SourcesProtoChecklist";
import { SourcesProtoDraftButton, SourcesProtoNoDraftLine, SourcesProtoRunLine } from "@/components/memory/prototype/SourcesProtoDraft";
import { SourcesProtoMemory } from "@/components/memory/prototype/SourcesProtoMemory";
import { SourcesProtoSourceList } from "@/components/memory/prototype/SourcesProtoSourceList";
import { sourcesMeta, useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import { cn } from "@/lib/utils";

// Sources as the first swimlane of the checklist, "Draft answers" in its header.
const SourcesLane = () => {
  const sources = useSourcesProtoStore((s) => s.sources);
  const folded = useSourcesProtoStore((s) => s.folds.sources ?? false);
  const toggle = useSourcesProtoStore((s) => s.toggleFold);
  const meta = sources.length === 0 ? "None yet" : sourcesMeta(sources);
  return (
    <section aria-label="Sources" className="space-y-2">
      <h3 className="flex items-center gap-2 border-b border-border pb-1">
        <button
          type="button"
          aria-expanded={!folded}
          onClick={() => toggle("sources", !folded)}
          className="group/lane flex min-w-0 flex-1 cursor-pointer items-center justify-between gap-2 rounded-sm px-1.5 py-1 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
        >
          <span className="min-w-0 truncate text-sm font-semibold">Sources</span>
          <span className="flex min-w-0 items-center gap-1 font-mono text-xs text-muted-foreground tabular-nums transition-colors group-hover/lane:text-foreground">
            <span className="truncate">{meta}</span>
            <ChevronDownIcon className={cn("size-3.5 shrink-0 transition-transform duration-150 ease-standard", folded && "-rotate-90")} aria-hidden />
          </span>
        </button>
        <SourcesProtoDraftButton />
      </h3>
      <div
        inert={folded}
        aria-hidden={folded}
        className={cn(
          "grid transition-[grid-template-rows,opacity] motion-reduce:transition-[opacity]",
          folded ? "[grid-template-rows:0fr] opacity-0 duration-150 ease-standard" : "[grid-template-rows:1fr] opacity-100 duration-200 ease-out",
        )}
      >
        <div className="min-h-0 space-y-2 overflow-hidden">
          <SourcesProtoRunLine className="px-1.5 py-1" />
          <SourcesProtoNoDraftLine className="px-1.5 py-1" />
          <SourcesProtoSourceList />
        </div>
      </div>
    </section>
  );
};

export const SourcesProtoVariantA = () => (
  <div className="@container">
    <div className="grid gap-12 @5xl:grid-cols-[minmax(0,36rem)_minmax(0,1fr)] @5xl:items-start @5xl:gap-10">
      <div className="min-w-0 space-y-6">
        <SourcesLane />
        <SourcesProtoChecklist />
      </div>
      <SourcesProtoMemory />
    </div>
  </div>
);
