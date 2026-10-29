import { useCallback, useRef, useState, type KeyboardEvent } from "react";
import { SearchIcon, SquarePlusIcon } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { CommandPaletteFooter } from "@/components/command/CommandPaletteFooter";
import { CommandPaletteGroup } from "@/components/command/CommandPaletteGroup";
import { Spinner } from "@/components/ui/spinner";
import { useCommandActionItems, useNewTicketRun } from "@/hooks/useCommandActionItems";
import { useCommandNavItems } from "@/hooks/useCommandNavItems";
import { useCommandSettingsItems } from "@/hooks/useCommandSettingsItems";
import { useCommandSearch } from "@/hooks/useCommandSearch";
import { useCommandPaletteStore } from "@/stores/commandPaletteStore";
import { arrangeCommandGroups, commandOptionId, type CommandGroup, type CommandItem } from "@/models/Command";

const LIST_ID = "command-palette-list";

// Rows shown at open appear with the palette; only rows that join the list after it fade in (index.css).
const settled = (el: HTMLDivElement) => requestAnimationFrame(() => (el.dataset.settled = ""));

// The kept scrollbar gutter comes out of the right padding, so classic scrollbars leave the rows evenly inset.
const measureGutter = (el: HTMLDivElement) => el.style.setProperty("--gutter", `${el.offsetWidth - el.clientWidth}px`);

// How many rows the typed list shows, or that a kind had more than it shows; nothing while browsing or with no results.
const footerStatus = (quiet: boolean, searching: boolean, cut: boolean, rows: number): string => {
  if (quiet || searching || rows === 0) return "";
  if (cut) return "More than shown · add a word";
  return `${rows} found`;
};

// Mounted only while the palette is open, so its queries run only then.
export const CommandPaletteBody = () => {
  const [query, setQuery] = useState("");
  const pick = useCommandPaletteStore((s) => s.pick);
  const [active, setActive] = useState(0);
  const list = useRef<HTMLDivElement>(null);
  const settle = useCallback((el: HTMLDivElement | null) => {
    list.current = el;
    if (!el) return;
    measureGutter(el);
    settled(el);
  }, []);
  const browsing = query.trim() === "";
  const nav = useCommandNavItems();
  const settings = useCommandSettingsItems(browsing);
  const actions = useCommandActionItems(browsing);
  const newTicket = useNewTicketRun();
  const { groups: found, searching, cut, found: hits } = useCommandSearch(query, true);

  const arranged = arrangeCommandGroups(browsing ? [...nav, ...actions] : [...nav, ...settings, ...actions], found, query);
  // Once a search of two characters or more has answered with nothing, Enter still does something: a ticket of that title.
  const nothing = arranged.length === 0 && hits !== undefined && !searching;
  const create: CommandGroup[] =
    nothing && newTicket
      ? [{ heading: "Create", quiet: true, items: [{ id: "create:ticket-titled", label: `New ticket “${query.trim()}”`, icon: SquarePlusIcon, run: () => newTicket(query.trim()) }] }]
      : [];
  const fresh = [...arranged, ...create];
  // A search still on its way keeps the last rows on screen, so the palette never collapses between two answers.
  const waiting = searching && fresh.length === 0;
  const freshKey = fresh.map((group) => group.items.map((item) => item.id).join()).join("|");
  const [shown, setShown] = useState({ key: "", groups: fresh });
  if (!waiting && shown.key !== freshKey) setShown({ key: freshKey, groups: fresh });
  const groups = waiting ? shown.groups : fresh;
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
  const announcement = searching || nothing || items.length === 0 ? "" : `${items.length} ${items.length === 1 ? "result" : "results"}`;

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
          aria-label="Search pages and everything in this workspace"
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
        className="command-results max-h-[min(26rem,60dvh)] overflow-y-auto overscroll-contain pr-[max(0px,calc(0.5rem-var(--gutter,0px)))] pb-1 pl-2"
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
      {nothing && <EmptyRow className="pt-1 pr-4 pb-4 pl-12 text-[13px] wrap-anywhere">Nothing titled or mentioning “{query.trim()}”</EmptyRow>}
      <p aria-live="polite" className="sr-only">
        {announcement}
      </p>
      <CommandPaletteFooter status={footerStatus(browsing || nothing, searching, cut, items.length)} open={items.length > 0} />
    </div>
  );
};
