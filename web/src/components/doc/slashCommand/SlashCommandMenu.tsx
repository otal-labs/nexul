import { forwardRef, useEffect, useImperativeHandle, useState } from "react";
import type { SuggestionKeyDownProps, SuggestionProps } from "@tiptap/suggestion";

import { cn } from "@/lib/utils";
import type { SlashCommandItem } from "@/components/doc/slashCommand/slashCommands";

export interface SlashCommandMenuRef {
  onKeyDown: (props: SuggestionKeyDownProps) => boolean;
}

// PlusMenuExtension just types "/" at the cursor rather than duplicating this popup.
export const SlashCommandMenu = forwardRef<SlashCommandMenuRef, SuggestionProps<SlashCommandItem>>(
  function SlashCommandMenu(props, ref) {
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
        className="glass max-h-64 w-56 overflow-y-auto rounded-lg p-1 text-popover-foreground shadow-overlay"
        role="listbox"
        aria-label="Insert block"
        data-testid="slash-command-menu"
      >
        {items.length === 0 && (
          <div className="px-3 py-1.5 text-sm text-muted-foreground">No matches</div>
        )}
        {items.map((item, index) => {
          const Icon = item.icon;
          return (
            <button
              key={item.id}
              type="button"
              role="option"
              aria-selected={index === selectedIndex}
              className={cn(
                "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm",
                index === selectedIndex && "bg-accent text-accent-foreground",
              )}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => selectItem(index)}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {item.label}
            </button>
          );
        })}
      </div>
    );
  },
);
