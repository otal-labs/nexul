import { Button } from "@/components/ui/button";
import { FromLine } from "@/components/memory/prototype/SourcesProtoQuestion";
import { valueLine } from "@/components/memory/prototype/SourcesProtoRow";
import type { ProtoDraft } from "@/components/memory/prototype/SourcesProtoData";
import { useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";

interface SourcesProtoSuggestionProps {
  id: string;
  progress: string;
  suggestion: ProtoDraft;
}

// An answered question a source now disagrees with: the answer as confirmed, then the suggestion and where it came from.
export const SourcesProtoSuggestion = ({ id, progress, suggestion }: SourcesProtoSuggestionProps) => {
  const answer = useSourcesProtoStore((s) => s.answers[id]);
  const accept = useSourcesProtoStore((s) => s.accept);
  const dismiss = useSourcesProtoStore((s) => s.dismiss);
  return (
    <div className="animate-in fade-in-0 slide-in-from-top-1 pr-1.5 pb-5 pl-9.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      <div className="mt-2 divide-y divide-border rounded-md border border-border">
        <div className="px-3 py-2">
          <p className="text-xs text-muted-foreground">Your answer</p>
          <p className="mt-0.5 text-sm">{valueLine(answer)}</p>
        </div>
        <div className="px-3 py-2">
          <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span aria-hidden className="size-1.5 rounded-full bg-warning" />
            Suggested change
          </p>
          <p className="mt-0.5 text-sm">{valueLine(suggestion.value)}</p>
          <FromLine from={suggestion.from} quote={suggestion.quote} />
        </div>
      </div>
      <div className="mt-4 flex items-center justify-end gap-2">
        <Button variant="ghost" size="sm" onClick={() => dismiss(id)}>
          Dismiss
        </Button>
        <Button size="sm" onClick={() => accept(id)}>
          Accept
        </Button>
      </div>
    </div>
  );
};
