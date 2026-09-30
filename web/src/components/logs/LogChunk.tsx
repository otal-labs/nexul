import { memo } from "react";

import { LogLineRow } from "@/components/logs/LogLineRow";
import type { ContainerLogLine } from "@/models/ContainerLog";

interface LogChunkProps {
  lines: ContainerLogLine[];
}

const sameLines = (a: LogChunkProps, b: LogChunkProps) =>
  a.lines.length === b.lines.length && a.lines.every((line, i) => line === b.lines[i]);

// A tick that appends or trims lines only changes the chunks at the two ends; the rest skip rendering.
export const LogChunk = memo(
  ({ lines }: LogChunkProps) => lines.map((line) => <LogLineRow key={line.seq} line={line} />),
  sameLines,
);
