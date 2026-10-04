import { LoaderCircle, Sparkles } from "lucide-react";

import { Button } from "@/components/ui/button";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { QUESTIONS } from "@/components/memory/prototype/SourcesProtoData";
import { followCount, useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import { cn } from "@/lib/utils";

// "Draft answers" once a follow source exists, with Regenerate's warning dot when a source changed since the last run.
export const SourcesProtoDraftButton = () => {
  const follow = useSourcesProtoStore((s) => followCount(s.sources));
  const run = useSourcesProtoStore((s) => s.run);
  const changed = useSourcesProtoStore((s) => s.sourcesChanged);
  const draft = useSourcesProtoStore((s) => s.draft);
  const stale = changed && run !== "drafting";
  if (follow === 0) return null;
  return (
    <Button variant="outline" size="sm" disabled={run === "drafting"} onClick={draft} className="shrink-0">
      {stale && <span role="img" aria-label="sources changed since the last drafting" className="size-2 rounded-full bg-warning" />}
      {run === "drafting" && <LoaderCircle className="size-3.5 animate-spin text-warning motion-reduce:animate-none" aria-hidden />}
      {run !== "drafting" && !stale && <Sparkles className="size-3.5" aria-hidden />}
      Draft answers
    </Button>
  );
};

// Shown where the button would be when every source is under question.
export const SourcesProtoNoDraftLine = ({ className }: { className?: string }) => {
  const show = useSourcesProtoStore((s) => s.sources.length > 0 && followCount(s.sources) === 0);
  if (!show) return null;
  return <p className={cn("text-sm text-muted-foreground", className)}>Sources under question are asked about in the follow-ups, not drafted from.</p>;
};

// The drafting run as one line, in the follow-up run's header-line style: the trail icon, the state, how many are drafted.
export const SourcesProtoRunLine = ({ className }: { className?: string }) => {
  const run = useSourcesProtoStore((s) => s.run);
  const drafted = useSourcesProtoStore((s) => s.drafted.length);
  if (run === "idle") return null;
  return (
    <p className={cn("flex min-w-0 items-center gap-2 text-sm", className)}>
      <TrailStateIcon state={run === "drafting" ? "running" : "done"} />
      <span className="shrink-0 font-medium">{run === "drafting" ? "Drafting" : "Drafts ready"}</span>
      <span className="truncate text-muted-foreground tabular-nums">
        · {drafted} of {QUESTIONS.length} drafted
      </span>
    </p>
  );
};
