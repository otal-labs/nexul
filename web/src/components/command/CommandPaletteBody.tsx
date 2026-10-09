import { useCallback, useRef, useState, type KeyboardEvent } from "react";
import { SearchIcon } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { CommandPaletteFooter } from "@/components/command/CommandPaletteFooter";
import { CommandPaletteGroup } from "@/components/command/CommandPaletteGroup";
import { Spinner } from "@/components/ui/spinner";
import { useCommandActionItems } from "@/hooks/useCommandActionItems";
import { useCommandNavItems } from "@/hooks/useCommandNavItems";
import { useCommandSearch } from "@/hooks/useCommandSearch";
import { useCommandPaletteStore } from "@/stores/commandPaletteStore";
import { commandOptionId, filterCommandGroups, type CommandItem } from "@/models/Command";

const LIST_ID = "command-palette-list";

// Rows shown at open appear with the palette; only rows that join the list after it fade in (index.css).
const settled = (el: HTMLDivElement) => requestAnimationFrame(() => (el.dataset.settled = ""));

// Mounted only while the palette is open, so its queries run only then.
export const CommandPaletteBody = () => {
  const [query, setQuery] = useState("");
  const pick = useCommandPaletteStore((s) => s.pick);
  const [active, setActive] = useState(0);
  const list = useRef<HTMLDivElement>(null);
  const settle = useCallback((el: HTMLDivElement | null) => {
    list.current = el;
    if (el) settled(el);
  }, []);
  const browsing = query.trim() === "";
  const nav = useCommandNavItems();
  const actions = useCommandActionItems(browsing);
  const { groups: found, searching } = useCommandSearch(query, true);

  // Browsing leads with what changed lately; typing leads with pages, then what the search found.
  const groups = filterCommandGroups(browsing ? [...found, ...nav, ...actions] : [...nav, ...found, ...actions], query);
  const items = groups.flatMap((group) => group.items);
  const activeIndex = Math.min(active, items.length - 1);
  const starts = groups.map((_, g) => groups.slice(0, g).reduce((sum, group) => sum + group.items.length, 0));

  const onSelect = (item: CommandItem) => {
    pick();
    item.run();
  };

  const move = (next: number) => {
    setActive(next);
    list.current?.querySelector(`[data-index="${next}"]`)?.scrollIntoView({ block: "nearest" });
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (items.length === 0) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      move((activeIndex + 1) % items.length);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      move((activeIndex - 1 + items.length) % items.length);
      return;
    }
    const item = items[activeIndex];
    if (event.key === "Enter" && item) {
      event.preventDefault();
      onSelect(item);
    }
  };

  // The empty sentence announces itself; otherwise the count is read once the search settles.
  const announcement = searching || items.length === 0 ? "" : `${items.length} ${items.length === 1 ? "result" : "results"}`;

  return (
    <div className="flex min-h-0 flex-col">
      <div className="flex items-center gap-3 border-b border-border px-4">
        <SearchIcon aria-hidden className="size-4 shrink-0 text-muted-foreground" />
        <input
          role="combobox"
          aria-expanded
          aria-controls={LIST_ID}
          aria-autocomplete="list"
          aria-activedescendant={items.length > 0 ? commandOptionId(activeIndex) : undefined}
          aria-label="Search pages, tickets, docs and memories"
          placeholder="Search or jump to…"
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          spellCheck={false}
          autoComplete="off"
          className="quiet-focus h-12 min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-muted-foreground"
        />
        {searching && <Spinner className="size-3.5 text-muted-foreground" />}
      </div>
      <div
        ref={settle}
        id={LIST_ID}
        role="listbox"
        aria-label="Results"
        className="command-results max-h-[min(26rem,60dvh)] overflow-y-auto overscroll-contain px-2 pb-1"
      >
        {groups.map((group, g) => (
          <CommandPaletteGroup
            key={group.heading}
            group={group}
            start={starts[g] ?? 0}
            activeIndex={activeIndex}
            onHover={setActive}
            onSelect={onSelect}
          />
        ))}
      </div>
      {items.length === 0 && !searching && <EmptyRow className="-mt-1 pt-0 wrap-anywhere">No results for “{query.trim()}”</EmptyRow>}
      <p aria-live="polite" className="sr-only">
        {announcement}
      </p>
      <CommandPaletteFooter />
    </div>
  );
};
