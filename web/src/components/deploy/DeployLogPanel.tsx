import { useEffect, useRef } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { microheaderClass } from "@/components/Microheader";
import { DeployLogActions } from "@/components/deploy/DeployLogActions";
import { fillToBottom } from "@/lib/fillToBottom";
import type { DeployLogLine } from "@/models/Stack";
import { formatLogTimestamp, phaseLabels } from "@/utils/DeployLogUtility";

interface DeployLogPanelProps {
  deployId: string;
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
      <span className="col-span-2 mt-2 mb-1 select-none text-micro tracking-[0.12em] text-muted-foreground uppercase in-[li:first-child]:mt-0">
        {phaseLabels[line.phase]}
      </span>
    )}
    <span className="select-none text-muted-foreground tabular-nums">{formatLogTimestamp(line.ts)}</span>
    {/* Compose pads its status lines with one space; deeper indentation is the output's own and stays. */}
    <span className="[overflow-wrap:anywhere] whitespace-pre-wrap">{line.text.replace(/^ (?=\S)/, "")}</span>
  </li>
);

// One well down to the panel's bottom: its own header bar holds what acts on it, and the lines fade under that bar once
// scrolled, so the tail it opens on never shows a line cut at a hard edge. Follows the tail while the reader sits at the
// bottom; scrolling up pins the view until they return.
export const DeployLogPanel = ({ deployId, lines, emptyMessage }: DeployLogPanelProps) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  // A line with only its timestamp says nothing; Copy and Download still carry every line.
  const shown = lines.filter((line) => line.text.trim() !== "");
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
    <div ref={fillToBottom} className="flex flex-col overflow-hidden rounded-lg border border-border bg-surface-2">
      <div className="flex shrink-0 items-center justify-between gap-3 border-b border-border py-1 pr-1 pl-3">
        <p className={microheaderClass}>
          Log
          {shown.length > 0 && <span className="ml-2 tabular-nums">{shown.length.toLocaleString()}</span>}
        </p>
        <DeployLogActions deployId={deployId} lines={lines} />
      </div>
      <div
        ref={scrollRef}
        onScroll={onScroll}
        role="log"
        aria-live="off"
        aria-label="Deploy log"
        tabIndex={0}
        className="scroll-edge min-h-0 flex-1 overflow-auto font-mono text-xs leading-6"
      >
        {shown.length === 0 && <EmptyRow className="border-0 font-sans">{emptyMessage}</EmptyRow>}
        {shown.length > 0 && (
          <ol className="px-3 py-2">
            {shown.map((line, i) => (
              <DeployLogRow key={line.seq} line={line} startsPhase={line.phase !== "" && line.phase !== shown[i - 1]?.phase} />
            ))}
          </ol>
        )}
      </div>
    </div>
  );
};
