import { CheckIcon, StarIcon } from "lucide-react";

import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { useModelFavouritesStore } from "@/stores/modelFavouritesStore";
import type { PickerEntry } from "@/models/ModelPick";
import { cn } from "@/lib/utils";

interface ModelPickerRowProps {
  id: string;
  entry: PickerEntry;
  selected: boolean;
  highlighted: boolean;
  shortcut: string | undefined;
  onPick: () => void;
  onHover: () => void;
}

// One row: the name with a New mark, the provider line under it, then the check, the shortcut, and a favourite star.
export const ModelPickerRow = ({ id, entry, selected, highlighted, shortcut, onPick, onHover }: ModelPickerRowProps) => {
  const { name, source, driver, isNew, favouriteKey } = entry;
  const favourite = useModelFavouritesStore((s) => !!favouriteKey && s.favourites.includes(favouriteKey));
  const toggle = useModelFavouritesStore((s) => s.toggle);
  return (
    <div role="presentation" className={cn("flex items-center gap-1 rounded-sm pr-1", highlighted && "bg-accent text-accent-foreground")}>
      <button
        id={id}
        type="button"
        role="option"
        aria-label={name}
        {...(source && { "aria-describedby": `${id}-source` })}
        aria-selected={selected}
        tabIndex={-1}
        // onMouseDown beats the search field's blur, keeping click-to-pick reliable.
        onMouseDown={(e) => {
          e.preventDefault();
          onPick();
        }}
        onMouseMove={onHover}
        className="flex min-w-0 flex-1 cursor-default items-center gap-2 py-1.5 pl-2 text-left outline-none"
      >
        <span className="min-w-0 flex-1">
          <span className="flex min-w-0 items-center gap-1.5">
            <span className="truncate text-sm">{name}</span>
            {isNew && (
              <span className="shrink-0 rounded-sm border border-border px-1 font-mono text-[10px] leading-4 text-muted-foreground uppercase">
                New
              </span>
            )}
          </span>
          {source && (
            <span className="mt-0.5 flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
              {driver && <HarnessProviderMark driver={driver} className="size-3 shrink-0" />}
              <span id={`${id}-source`} className="truncate">
                {source}
              </span>
            </span>
          )}
        </span>
        {selected && <CheckIcon className="size-3.5 shrink-0" aria-hidden />}
        {shortcut && <kbd className="shrink-0 font-mono text-[10px] text-muted-foreground">{shortcut}</kbd>}
      </button>
      {favouriteKey && (
        <button
          type="button"
          aria-label={favourite ? `Remove ${name} from favourites` : `Add ${name} to favourites`}
          aria-pressed={favourite}
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => toggle(favouriteKey)}
          className="shrink-0 rounded-sm p-1 text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
        >
          <StarIcon className={cn("size-3.5", favourite && "fill-current text-foreground")} aria-hidden />
        </button>
      )}
    </div>
  );
};
