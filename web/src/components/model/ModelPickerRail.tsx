import { StarIcon } from "lucide-react";
import type { ReactNode } from "react";

import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import type { RailFilter } from "@/models/ModelPick";
import type { HarnessProvider } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface RailButtonProps {
  label: string;
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}

const RailButton = ({ label, active, onClick, children }: RailButtonProps) => (
  <button
    type="button"
    aria-label={label}
    title={label}
    aria-pressed={active}
    onClick={onClick}
    className={cn(
      "flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40 hover:text-foreground",
      active && "bg-accent text-foreground",
    )}
  >
    {children}
  </button>
);

interface ModelPickerRailProps {
  providers: HarnessProvider[];
  filter: RailFilter;
  onFilter: (filter: RailFilter) => void;
}

// Favourites, then one button per provider; pressing the active one goes back to every model.
export const ModelPickerRail = ({ providers, filter, onFilter }: ModelPickerRailProps) => {
  const toggle = (next: RailFilter, active: boolean) => onFilter(active ? { kind: "all" } : next);
  const favouritesActive = filter.kind === "favourites";
  return (
    <div role="toolbar" aria-label="Filter models" aria-orientation="vertical" className="flex flex-col gap-1 border-r border-border p-1">
      <RailButton label="Favourites" active={favouritesActive} onClick={() => toggle({ kind: "favourites" }, favouritesActive)}>
        <StarIcon className="size-3.5" aria-hidden />
      </RailButton>
      {providers.map((p) => (
        <RailButton
          key={p.id}
          label={p.name}
          active={filter.kind === "provider" && filter.id === p.id}
          onClick={() => toggle({ kind: "provider", id: p.id }, filter.kind === "provider" && filter.id === p.id)}
        >
          <HarnessProviderMark driver={p.driver} className="size-3.5" />
        </RailButton>
      ))}
    </div>
  );
};
