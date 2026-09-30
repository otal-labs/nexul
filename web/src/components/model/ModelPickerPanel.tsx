import { useId, useMemo, useState, type KeyboardEvent } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { ModelPickerRail } from "@/components/model/ModelPickerRail";
import { ModelPickerRow } from "@/components/model/ModelPickerRow";
import { useModelFavouritesStore } from "@/stores/modelFavouritesStore";
import { pickerEntries, type ModelPick, type PickerEntry, type RailFilter } from "@/models/ModelPick";
import type { HarnessProvider } from "@/models/Pairing";

const SHORTCUT_KEY = /Mac|iPhone|iPad/.test(navigator.userAgent) ? "⌘" : "Ctrl ";

interface ModelPickerPanelProps {
  providers: HarnessProvider[];
  value: ModelPick;
  allowDefault: boolean;
  label: string;
  onPick: (pick: ModelPick) => void;
}

// The picker's popover body: the provider rail, the search field, and the rows, the legacy ones grouped last.
export const ModelPickerPanel = ({ providers, value, allowDefault, label, onPick }: ModelPickerPanelProps) => {
  const favourites = useModelFavouritesStore((s) => s.favourites);
  const [filter, setFilter] = useState<RailFilter>({ kind: "all" });
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const listId = useId();
  const entries = useMemo(
    () => pickerEntries(providers, filter, query, favourites, allowDefault),
    [providers, filter, query, favourites, allowDefault],
  );
  const current = entries.filter((e) => !e.legacy);
  const legacy = entries.filter((e) => e.legacy);

  const move = (index: number) => {
    setHighlighted(index);
    document.getElementById(`${listId}-${index}`)?.scrollIntoView({ block: "nearest" });
  };
  const pickAt = (index: number) => {
    const entry = entries[index];
    if (entry) onPick(entry.pick);
  };
  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    const digit = Number(e.key);
    if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter" || ((e.metaKey || e.ctrlKey) && digit >= 1)) {
      e.preventDefault();
    }
    if (e.key === "ArrowDown") move(Math.min(highlighted + 1, entries.length - 1));
    if (e.key === "ArrowUp") move(Math.max(highlighted - 1, 0));
    if (e.key === "Enter") pickAt(highlighted);
    if ((e.metaKey || e.ctrlKey) && digit >= 1) pickAt(digit - 1);
  };
  const row = (entry: PickerEntry, index: number) => (
    <ModelPickerRow
      key={entry.key}
      id={`${listId}-${index}`}
      entry={entry}
      selected={entry.pick.provider === value.provider && entry.pick.model === value.model}
      highlighted={index === highlighted}
      shortcut={index < 9 ? `${SHORTCUT_KEY}${index + 1}` : undefined}
      onPick={() => onPick(entry.pick)}
      onHover={() => setHighlighted(index)}
    />
  );

  return (
    <div className="flex max-h-96">
      {providers.length > 1 && (
        <ModelPickerRail
          providers={providers}
          filter={filter}
          onFilter={(next) => {
            setFilter(next);
            setHighlighted(0);
          }}
        />
      )}
      <div className="flex min-w-0 flex-1 flex-col p-1">
        <input
          // The popover just opened at the user's request; focus belongs in the search.
          autoFocus
          role="combobox"
          aria-expanded
          aria-controls={listId}
          aria-activedescendant={entries.length > 0 ? `${listId}-${highlighted}` : undefined}
          aria-label={`Search ${label.toLowerCase()}`}
          placeholder="Search models…"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setHighlighted(0);
          }}
          onKeyDown={onKeyDown}
          className="mb-1 h-8 w-full shrink-0 rounded-sm border border-input bg-transparent px-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring"
        />
        <div id={listId} role="listbox" aria-label={label} className="min-h-0 overflow-y-auto">
          {current.map((entry, index) => row(entry, index))}
          {legacy.length > 0 && (
            <div role="group" aria-labelledby={`${listId}-legacy`}>
              <p id={`${listId}-legacy`} className="px-2 pt-2 pb-1 font-mono text-[11px] text-muted-foreground uppercase">
                Legacy models
              </p>
              {legacy.map((entry, index) => row(entry, current.length + index))}
            </div>
          )}
          {entries.length === 0 && <EmptyRow className="border-0 px-2 py-4">No models match.</EmptyRow>}
        </div>
      </div>
    </div>
  );
};
