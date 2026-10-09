import { useEffect, useRef } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import type { DeployLogLine } from "@/models/Stack";
import { formatLogTimestamp, phaseLabels } from "@/utils/DeployLogUtility";

interface DeployLogPanelProps {
  lines: DeployLogLine[];
  emptyMessage: string;
}

const FOLLOW_SLACK_PX = 8;

interface DeployLogRowProps {
  line: DeployLogLine;
  startsPhase: boolean;
}

// The first line of each phase carries the step's name above it, so the log reads against the timeline.
const DeployLogRow = ({ line, startsPhase }: DeployLogRowProps) => (
  <li className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3">
    {startsPhase && line.phase !== "" && (
      <span className="col-span-2 mt-2 mb-1 select-none text-[11px] tracking-[0.12em] text-muted-foreground uppercase in-[li:first-child]:mt-0">
        {phaseLabels[line.phase]}
      </span>
    )}
    <span className="select-none text-muted-foreground tabular-nums">{formatLogTimestamp(line.ts)}</span>
    <span className="[overflow-wrap:anywhere] whitespace-pre-wrap">{line.text}</span>
  </li>
);

// Follows the tail while the user sits at the bottom; scrolling up pins the view until they return.
export const DeployLogPanel = ({ lines, emptyMessage }: DeployLogPanelProps) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  const followRef = useRef(true);

  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    followRef.current = el.scrollHeight - el.scrollTop - el.clientHeight <= FOLLOW_SLACK_PX;
  };

  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !followRef.current) return;
    el.scrollTop = el.scrollHeight;
  }, [lines.length]);

  return (
    <div
      ref={scrollRef}
      onScroll={onScroll}
      role="log"
      aria-live="off"
      aria-label="Deploy log"
      tabIndex={0}
      className="h-72 overflow-auto rounded-lg border border-border bg-surface-2 font-mono text-xs leading-6 sm:h-96"
    >
      {lines.length === 0 && <EmptyRow className="border-0 font-sans">{emptyMessage}</EmptyRow>}
      {lines.length > 0 && (
        <ol className="px-3 py-2">
          {lines.map((line, i) => (
            <DeployLogRow key={line.seq} line={line} startsPhase={line.phase !== "" && line.phase !== lines[i - 1]?.phase} />
          ))}
        </ol>
      )}
    </div>
  );
};
