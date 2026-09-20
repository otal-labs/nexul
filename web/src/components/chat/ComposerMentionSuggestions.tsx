import type { MentionCandidate } from "@/models/Chat";
import { cn } from "@/lib/utils";

interface ComposerMentionSuggestionsProps {
  matches: MentionCandidate[];
  selectedIndex: number;
  onPick: (handle: string) => void;
}

export const ComposerMentionSuggestions = ({ matches, selectedIndex, onPick }: ComposerMentionSuggestionsProps) => (
  <div
    role="listbox"
    aria-label="Mention suggestions"
    className="absolute inset-x-2 bottom-full z-10 mb-1 max-h-48 overflow-y-auto rounded-md border border-border bg-popover p-1 shadow-overlay"
  >
    {matches.map((candidate, index) => (
      <button
        key={candidate.handle}
        type="button"
        role="option"
        aria-selected={index === selectedIndex}
        onMouseDown={(event) => event.preventDefault()}
        onClick={() => onPick(candidate.handle)}
        className={cn(
          "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm",
          index === selectedIndex && "bg-accent",
        )}
      >
        @{candidate.handle}
        {candidate.kind === "agent" && <span className="ml-auto text-[10px] text-muted-foreground">Agent</span>}
      </button>
    ))}
  </div>
);
