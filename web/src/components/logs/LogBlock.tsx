import { ArrowDownIcon } from "lucide-react";

import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller";
import { LogChunk } from "@/components/logs/LogChunk";
import { fillToBottom } from "@/lib/fillToBottom";
import type { ContainerLogLine } from "@/models/ContainerLog";
import { chunkLogLines, type EmptyLogMessage } from "@/utils/ContainerLogUtility";

interface LogBlockProps {
  lines: ContainerLogLine[];
  empty: EmptyLogMessage;
}

// One terminal-style block: the scroller follows the newest line and lets go the moment the reader scrolls up.
export const LogBlock = ({ lines, empty }: LogBlockProps) => (
  <div ref={fillToBottom} className="relative overflow-hidden rounded-lg border border-border bg-surface-2">
    {lines.length === 0 && (
      <div className="flex h-full flex-col items-center justify-center px-6 text-center">
        <p role="status" className="max-w-md text-sm text-balance text-muted-foreground">{empty.text}</p>
        {empty.detail && (
          <details className="mt-2 max-w-md text-xs text-muted-foreground">
            <summary className="mx-auto w-fit cursor-pointer hover:text-foreground">Details</summary>
            <p className="mt-1 font-mono break-words">{empty.detail}</p>
          </details>
        )}
      </div>
    )}
    {lines.length > 0 && (
      <MessageScroller className="h-full">
        <MessageScrollerViewport aria-label="Container log" className="scroll-edge py-2">
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
  </div>
);
