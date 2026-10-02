import { MessageSquare } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import type { ThreadVariant } from "@/components/ticket/threadVariants";
import { Button } from "@/components/ui/button";

interface ThreadStartPromptProps {
  empty: ThreadVariant["empty"];
  onStart: () => void;
}

// Prototype only: three readings of a ticket with no thread yet, one per candidate column.
export const ThreadStartPrompt = ({ empty, onStart }: ThreadStartPromptProps) => (
  <div className="space-y-3">
    {empty === "sentence" && <p className="text-sm text-muted-foreground">No messages yet.</p>}
    {empty === "sentence" && (
      <Button variant="outline" size="sm" className="w-full" onClick={onStart}>
        <MessageSquare className="size-4" aria-hidden />
        Start chat
      </Button>
    )}
    {empty === "card" && (
      <EmptyState
        size="compact"
        icon={MessageSquare}
        title="No thread yet"
        message="Ask the team, or mention @Agent to put it to work on this ticket."
        action={
          <Button variant="outline" size="sm" onClick={onStart}>
            Start chat
          </Button>
        }
      />
    )}
    {empty === "composer" && (
      <button
        type="button"
        onClick={onStart}
        className="flex h-9 w-full items-center gap-2 rounded-md border border-input bg-transparent px-3 text-left text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:border-ring/40 hover:text-foreground"
      >
        <MessageSquare className="size-4 shrink-0" aria-hidden />
        <span className="truncate">Message this ticket…</span>
      </button>
    )}
  </div>
);
