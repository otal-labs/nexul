import { useEffect, useRef } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import type { DeployLogLine } from "@/models/Stack";
import { formatLogTimestamp } from "@/utils/DeployLogUtility";

interface DeployLogPanelProps {
  lines: DeployLogLine[];
  emptyMessage: string;
}

const FOLLOW_SLACK_PX = 8;

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
        <ol className="w-max min-w-full px-3 py-2">
          {lines.map((line) => (
            <li key={line.seq} className="flex gap-3 whitespace-pre">
              <span className="shrink-0 select-none text-muted-foreground">{formatLogTimestamp(line.ts)}</span>
              <span>{line.text}</span>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
};
