import { X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { InterviewSourceKindIcon } from "@/components/memory/InterviewSourceKindIcon";
import { InterviewSourceStance } from "@/components/memory/InterviewSourceStance";
import { useRemoveInterviewSource, useSetInterviewSourceStance } from "@/hooks/InterviewSourceHooks";
import { SOURCE_KIND_LABEL, SOURCE_STANCE_LABEL, isNamed, sourceName, type InterviewSource } from "@/models/InterviewSource";
import { cn } from "@/lib/utils";

interface InterviewSourceRowProps {
  source: InterviewSource;
  // Added, or its content changed, since the last drafting run.
  change: "new" | "changed" | null;
  readOnly: boolean;
}

// One source as a hairline row: kind icon, name, stance, remove; a gone source keeps only Remove.
export const InterviewSourceRow = ({ source, change, readOnly }: InterviewSourceRowProps) => {
  const setStance = useSetInterviewSourceStance(source.project_id);
  const remove = useRemoveInterviewSource(source.project_id);
  const name = sourceName(source);
  const named = isNamed(source);
  return (
    <li className="flex min-h-11 items-center gap-2.5 border-b border-border px-1.5 py-1.5">
      <InterviewSourceKindIcon kind={source.kind} name={source.label} />
      <span className="flex min-w-0 flex-1 items-baseline gap-2">
        {source.gone && <span className="truncate text-sm text-muted-foreground">{SOURCE_KIND_LABEL[source.kind]} · No longer there</span>}
        {source.not_visible && <span className="truncate text-sm text-muted-foreground">{SOURCE_KIND_LABEL[source.kind]} · Not visible to you</span>}
        {named && (
          <span className={cn("truncate", source.kind === "path" ? "font-mono text-sm" : "text-sm")} title={name}>
            {name}
          </span>
        )}
        {change !== null && <span className="shrink-0 font-mono text-xs text-muted-foreground">{change}</span>}
      </span>
      {readOnly && !source.gone && <span className="shrink-0 font-mono text-xs text-muted-foreground">{SOURCE_STANCE_LABEL[source.stance]}</span>}
      {!readOnly && !source.gone && (
        <InterviewSourceStance
          label={name}
          value={source.stance}
          disabled={source.not_visible ?? false}
          onChange={(stance) => stance !== source.stance && setStance.mutate({ id: source.id, stance, was: source.stance })}
        />
      )}
      {!readOnly && source.gone && (
        <Button variant="ghost" size="sm" className="h-7 shrink-0 px-2 text-xs" loading={remove.isPending} onClick={() => remove.mutate(source.id)}>
          Remove
        </Button>
      )}
      {!readOnly && !source.gone && (
        <Button
          variant="ghost"
          size="icon"
          className="size-7 shrink-0 text-muted-foreground"
          aria-label={`Remove ${name}`}
          disabled={remove.isPending}
          onClick={() => remove.mutate(source.id)}
        >
          <X className="size-3.5" />
        </Button>
      )}
    </li>
  );
};
