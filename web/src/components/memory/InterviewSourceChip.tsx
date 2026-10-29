import type { ReactNode } from "react";
import { Link } from "react-router";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { InterviewPathName } from "@/components/memory/InterviewPathName";
import { InterviewSourceKindIcon } from "@/components/memory/InterviewSourceKindIcon";
import { useFetchMemories } from "@/hooks/MemoryHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useInterviewSourceFocusStore } from "@/stores/interviewSourceFocusStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { isRepositoryRoot, sourceName, type InterviewSource } from "@/models/InterviewSource";
import { memoryPath } from "@/models/Project";
import { cn } from "@/lib/utils";

interface InterviewSourceChipProps {
  source: InterviewSource;
}

const chipClass =
  "inline-flex h-6 max-w-full min-w-0 cursor-pointer items-center gap-1.5 rounded-full border border-border px-2.5 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40 hover:text-foreground";

const useFocusHandlers = (sourceId: string) => {
  const setSourceId = useInterviewSourceFocusStore((s) => s.setSourceId);
  const on = () => setSourceId(sourceId);
  const off = () => setSourceId(null);
  return { onMouseEnter: on, onFocus: on, onMouseLeave: off, onBlur: off };
};

const ChipFace = ({ source }: InterviewSourceChipProps) => (
  <>
    <InterviewSourceKindIcon kind={source.kind} name={source.label} className="size-3.5" />
    {source.kind === "path" && !isRepositoryRoot(source) && <InterviewPathName path={sourceName(source)} />}
    {(source.kind !== "path" || isRepositoryRoot(source)) && <span className="truncate">{sourceName(source)}</span>}
  </>
);

const MemoryChip = ({ source, handlers }: InterviewSourceChipProps & { handlers: ReturnType<typeof useFocusHandlers> }) => {
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: memories } = useFetchMemories(workspaceId);
  const projectId = memories?.find((m) => m.id === source.ref)?.project_id ?? source.project_id;
  return (
    <Link to={wsPath(memoryPath(projectId, source.ref))} className={chipClass} {...handlers}>
      <ChipFace source={source} />
    </Link>
  );
};

const LinkChip = ({ to, children, handlers }: { to: string; children: ReactNode; handlers: ReturnType<typeof useFocusHandlers> }) => {
  const wsPath = useWorkspacePath();
  return (
    <Link to={wsPath(to)} className={chipClass} {...handlers}>
      {children}
    </Link>
  );
};

// One source a draft names, as a neutral chip that opens it: a doc, memory or project goes to its page, pasted text opens in a
// popover, and a path, which Nexul cannot open, names itself in full in a tooltip.
export const InterviewSourceChip = ({ source }: InterviewSourceChipProps) => {
  const handlers = useFocusHandlers(source.id);
  const face = <ChipFace source={source} />;
  return (
    <>
      {source.kind === "doc" && <LinkChip to={`/docs/${source.ref}`} handlers={handlers}>{face}</LinkChip>}
      {source.kind === "project" && <LinkChip to={`/board/${source.ref}`} handlers={handlers}>{face}</LinkChip>}
      {source.kind === "memory" && <MemoryChip source={source} handlers={handlers} />}
      {source.kind === "text" && (
        <Popover>
          <PopoverTrigger className={chipClass} {...handlers}>
            {face}
          </PopoverTrigger>
          <PopoverContent align="start" className="max-h-72 w-96 overflow-y-auto text-sm whitespace-pre-wrap">
            <p className="mb-2 font-medium">{sourceName(source)}</p>
            <p className="text-muted-foreground">{source.body}</p>
          </PopoverContent>
        </Popover>
      )}
      {source.kind === "path" && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span tabIndex={0} className={cn(chipClass, "cursor-default")} {...handlers}>
              {face}
            </span>
          </TooltipTrigger>
          <TooltipContent className={isRepositoryRoot(source) ? undefined : "font-mono"}>{sourceName(source)}</TooltipContent>
        </Tooltip>
      )}
    </>
  );
};
