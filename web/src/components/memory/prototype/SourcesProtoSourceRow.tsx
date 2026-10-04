import { X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { KIND_LABEL, SourceKindIcon, SourceStance } from "@/components/memory/prototype/SourcesProtoStance";
import type { ProtoSource } from "@/components/memory/prototype/SourcesProtoData";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import { cn } from "@/lib/utils";

interface SourceNameProps {
  source: ProtoSource;
}

// What a source is called, or why it can't be named: gone, or pointing at something this viewer cannot read.
export const SourceName = ({ source }: SourceNameProps) => (
  <span className="flex min-w-0 flex-1 items-baseline gap-2">
    {source.gone && <span className="truncate text-sm text-muted-foreground">{KIND_LABEL[source.kind]} · No longer there</span>}
    {source.hidden && <span className="truncate text-sm text-muted-foreground">{KIND_LABEL[source.kind]} · Not visible to you</span>}
    {!source.gone && !source.hidden && (
      <span className={cn("truncate", source.kind === "path" ? "font-mono text-[13px]" : "text-sm")} title={source.name}>
        {source.name}
      </span>
    )}
    {source.fresh && <span className="shrink-0 font-mono text-[10px] text-muted-foreground uppercase">New</span>}
  </span>
);

export const sourceLabel = (source: ProtoSource) => (source.gone || source.hidden ? KIND_LABEL[source.kind] : source.name);

// One source as a hairline row: kind icon, name, stance, remove; a gone source keeps only Remove.
export const SourcesProtoSourceRow = ({ source }: { source: ProtoSource }) => {
  const setStance = useSourcesProtoStore((s) => s.setStance);
  const remove = useSourcesProtoStore((s) => s.removeSource);
  return (
    <li className="flex min-h-11 items-center gap-2.5 border-b border-border px-1.5 py-1.5">
      <SourceKindIcon source={source} />
      <SourceName source={source} />
      {!source.gone && (
        <SourceStance label={sourceLabel(source)} value={source.stance} onChange={(v) => setStance(source.id, v)} disabled={source.hidden ?? false} />
      )}
      {source.gone && (
        <Button variant="ghost" size="sm" className="h-7 shrink-0 px-2 text-xs" onClick={() => remove(source.id)}>
          Remove
        </Button>
      )}
      {!source.gone && (
        <Button variant="ghost" size="icon" className="size-7 shrink-0 text-muted-foreground" aria-label={`Remove ${sourceLabel(source)}`} onClick={() => remove(source.id)}>
          <X className="size-3.5" />
        </Button>
      )}
    </li>
  );
};
