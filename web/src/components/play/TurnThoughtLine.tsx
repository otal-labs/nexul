import { Brain } from "lucide-react";

import type { ActivityEntry } from "@/models/Trail";

interface TurnThoughtLineProps {
  entry: ActivityEntry;
}

// A thought the Agent summarised between its steps, muted and in italics so it never reads as something it said.
export const TurnThoughtLine = ({ entry }: TurnThoughtLineProps) => (
  <p className="flex items-start gap-2 px-1 py-0.5 text-row leading-5 text-muted-foreground italic">
    <span className="flex h-5 shrink-0 items-center">
      <Brain className="size-3.5" aria-hidden />
    </span>
    <span className="min-w-0 break-words whitespace-pre-wrap">
      <span className="sr-only">Thought: </span>
      {entry.detail || entry.summary}
    </span>
  </p>
);
