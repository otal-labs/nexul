import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SourcesProtoAddForm } from "@/components/memory/prototype/SourcesProtoAddForm";
import { SourcesProtoSourceRow } from "@/components/memory/prototype/SourcesProtoSourceRow";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";

export const EMPTY_SOURCES = "Point the interview at what this project already has: a practices folder, a standards doc, the project it replaces.";

// The sources as hairline rows, then the inline add form or the button that opens it.
export const SourcesProtoSourceList = () => {
  const sources = useSourcesProtoStore((s) => s.sources);
  const adding = useSourcesProtoStore((s) => s.adding);
  const setAdding = useSourcesProtoStore((s) => s.setAdding);
  return (
    <div className="space-y-3">
      {sources.length === 0 && !adding && <p className="px-1.5 text-sm text-muted-foreground">{EMPTY_SOURCES}</p>}
      {sources.length > 0 && (
        <ul>
          {sources.map((source) => (
            <SourcesProtoSourceRow key={source.id} source={source} />
          ))}
        </ul>
      )}
      {adding && <SourcesProtoAddForm />}
      {!adding && (
        <Button variant="ghost" size="sm" onClick={() => setAdding("path")} className="text-muted-foreground">
          <Plus className="size-3.5" />
          Add source
        </Button>
      )}
    </div>
  );
};
