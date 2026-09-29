import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

export type EnvMode = "fields" | "paste";

interface EnvModeToggleProps {
  mode: EnvMode;
  onMode: (mode: EnvMode) => void;
  // Switching to Fields would drop lines that are not KEY=value, so it waits until they are fixed.
  fieldsDisabled: boolean;
}

export const EnvModeToggle = ({ mode, onMode, fieldsDisabled }: EnvModeToggleProps) => (
  <ToggleGroup
    type="single"
    variant="outline"
    size="sm"
    aria-label="Environment input"
    value={mode}
    onValueChange={(next) => next && onMode(next as EnvMode)}
    className="ml-auto"
  >
    <ToggleGroupItem value="fields" disabled={fieldsDisabled} className="h-7 px-3 text-xs">
      Fields
    </ToggleGroupItem>
    <ToggleGroupItem value="paste" className="h-7 px-3 text-xs">
      Paste .env
    </ToggleGroupItem>
  </ToggleGroup>
);
