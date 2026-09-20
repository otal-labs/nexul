import { forwardRef, useEffect, useImperativeHandle, useState } from "react";
import { FileTextIcon, Loader2Icon, TicketIcon } from "lucide-react";
import type { SuggestionKeyDownProps, SuggestionProps } from "@tiptap/suggestion";

import { cn } from "@/lib/utils";
import type { MentionSearchResult } from "@/models/Mention";

export interface MentionSuggestionsRef {
  onKeyDown: (props: SuggestionKeyDownProps) => boolean;
}

interface MentionSuggestionRowProps {
  item: MentionSearchResult;
  selected: boolean;
  onSelect: () => void;
}

// One entry of the @ picker (F1: named row component, not an inline list body).
const MentionSuggestionRow = ({ item, selected, onSelect }: MentionSuggestionRowProps) => {
  const Icon = item.type === "ticket" ? TicketIcon : FileTextIcon;
  return (
    <button
      type="button"
      role="option"
      aria-selected={selected}
      className={cn(
        "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm",
        selected && "bg-accent text-accent-foreground",
      )}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onSelect}
    >
      <Icon className="h-4 w-4 shrink-0" />
      <span className="min-w-0 truncate">{item.title}</span>
      {item.type === "ticket" && item.status_label && (
        <span className="shrink-0 rounded-full bg-secondary px-2 py-0.5 text-xs text-secondary-foreground">
          {item.status_label}
        </span>
      )}
      <span className="ml-auto shrink-0 text-xs text-muted-foreground">
        {item.type === "ticket" ? "ticket" : "doc"}
      </span>
    </button>
  );
};

// Own React root; items arrive pre-resolved from the suggestion plugin, not the app's query cache.
export const MentionSuggestions = forwardRef<
  MentionSuggestionsRef,
  SuggestionProps<MentionSearchResult>
>(function MentionSuggestions(props, ref) {
  const items = props.items;
  const [selectedIndex, setSelectedIndex] = useState(0);

  useEffect(() => {
    setSelectedIndex(0);
  }, [items]);

  const selectItem = (index: number) => {
    const item = items[index];
    if (item) props.command(item);
  };

  useImperativeHandle(ref, () => ({
    onKeyDown: ({ event }) => {
      if (items.length === 0) return false;
      if (event.key === "ArrowUp") {
        setSelectedIndex((i) => (i + items.length - 1) % items.length);
        return true;
      }
      if (event.key === "ArrowDown") {
        setSelectedIndex((i) => (i + 1) % items.length);
        return true;
      }
      if (event.key === "Enter") {
        selectItem(selectedIndex);
        return true;
      }
      return false;
    },
  }));

  return (
    <div
      className="glass max-h-64 w-72 overflow-y-auto rounded-lg p-1 text-popover-foreground shadow-overlay"
      role="listbox"
      aria-label="Mention suggestions"
      data-testid="mention-suggestions"
    >
      {props.loading && items.length === 0 && (
        <div className="flex items-center gap-2 px-3 py-1.5 text-sm text-muted-foreground">
          <Loader2Icon className="h-4 w-4 animate-spin" />
          Searching…
        </div>
      )}
      {!props.loading && items.length === 0 && (
        <div className="px-3 py-1.5 text-sm text-muted-foreground">No matches</div>
      )}
      {items.map((item, index) => (
        <MentionSuggestionRow
          key={`${item.type}:${item.id}`}
          item={item}
          selected={index === selectedIndex}
          onSelect={() => selectItem(index)}
        />
      ))}
    </div>
  );
});
