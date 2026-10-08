import { ArrowDownIcon, CopyIcon } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { Button } from "@/components/ui/button";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller";
import { LogChunk } from "@/components/logs/LogChunk";
import type { ContainerLogLine } from "@/models/ContainerLog";
import { chunkLogLines, type EmptyLogMessage } from "@/utils/ContainerLogUtility";

interface LogBlockProps {
  lines: ContainerLogLine[];
  empty: EmptyLogMessage;
  onCopy: () => void;
}

// One terminal-style block: the scroller follows the newest line and lets go the moment the reader scrolls up.
export const LogBlock = ({ lines, empty, onCopy }: LogBlockProps) => (
  <div className="relative overflow-hidden rounded-lg border border-border bg-surface-2">
    {lines.length === 0 && (
      <div className="h-96 px-4 py-32">
        <EmptyRow className="p-0">{empty.text}</EmptyRow>
        {empty.detail && (
          <details className="mt-2 text-xs text-muted-foreground">
            <summary className="w-fit cursor-pointer hover:text-foreground">Details</summary>
            <p className="mt-1 font-mono break-words">{empty.detail}</p>
          </details>
        )}
      </div>
    )}
    {lines.length > 0 && (
      <MessageScroller className="h-[34rem] max-h-[70vh]">
        <MessageScrollerViewport aria-label="Container log" className="py-2">
          <MessageScrollerContent className="gap-0">
            {chunkLogLines(lines).map(([bucket, chunk]) => (
              <MessageScrollerItem key={bucket} style={{ containIntrinsicSize: `auto ${chunk.length * 20}px` }}>
                <LogChunk lines={chunk} />
              </MessageScrollerItem>
            ))}
          </MessageScrollerContent>
        </MessageScrollerViewport>
        <MessageScrollerButton size="sm" variant="secondary" className="w-auto bg-card font-sans text-xs hover:bg-accent">
          <ArrowDownIcon aria-hidden /> Jump to live
        </MessageScrollerButton>
      </MessageScroller>
    )}
    <Button
      type="button"
      variant="outline"
      size="icon"
      onClick={onCopy}
      disabled={lines.length === 0}
      aria-label="Copy log"
      className="absolute top-2 right-4 size-8 bg-background/80"
    >
      <CopyIcon aria-hidden />
    </Button>
  </div>
);
