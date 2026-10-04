import { X } from "lucide-react";

import { SourceKindIcon, SourceStance } from "@/components/memory/prototype/SourcesProtoStance";
import { SourceName, sourceLabel } from "@/components/memory/prototype/SourcesProtoSourceRow";
import type { ProtoSource } from "@/components/memory/prototype/SourcesProtoData";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";

// One source in the page strip: a hairline box of kind icon, name, stance, and remove.
export const SourcesProtoSourceChip = ({ source }: { source: ProtoSource }) => {
  const setStance = useSourcesProtoStore((s) => s.setStance);
  const remove = useSourcesProtoStore((s) => s.removeSource);
  return (
    <li className="flex h-9 max-w-[min(100%,24rem)] min-w-0 items-center gap-2 rounded-md border border-border bg-card pr-1 pl-2.5">
      <SourceKindIcon source={source} className="size-3.5" />
      <SourceName source={source} />
      {!source.gone && (
        <SourceStance label={sourceLabel(source)} value={source.stance} onChange={(v) => setStance(source.id, v)} disabled={source.hidden ?? false} />
      )}
      {source.gone && (
        <button type="button" onClick={() => remove(source.id)} className="shrink-0 rounded-sm px-1.5 text-xs hover:bg-accent">
          Remove
        </button>
      )}
      {!source.gone && (
        <button
          type="button"
          aria-label={`Remove ${sourceLabel(source)}`}
          onClick={() => remove(source.id)}
          className="flex size-6 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <X className="size-3.5" />
        </button>
      )}
    </li>
  );
};
