import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { SOURCE_STANCE_LABEL, SOURCE_STANCES, type SourceStance } from "@/models/InterviewSource";
import { cn } from "@/lib/utils";

interface InterviewSourceStanceProps {
  // Names the group for a screen reader: "Stance for <label>".
  label: string;
  value: SourceStance;
  onChange: (value: SourceStance) => void;
  disabled?: boolean;
  size?: "xs" | "sm";
}

const itemClass = { xs: "h-6 px-2 text-xs", sm: "h-7 px-3 text-xs" };

// Follow | Question as a segmented pair.
export const InterviewSourceStance = ({ label, value, onChange, disabled = false, size = "xs" }: InterviewSourceStanceProps) => (
  <ToggleGroup
    type="single"
    variant="outline"
    size="sm"
    aria-label={`Stance for ${label}`}
    disabled={disabled}
    value={value}
    // Clicking the selected side reports it again, so a preselected default can be picked on purpose.
    onValueChange={(next) => onChange((next || value) as SourceStance)}
    className="shrink-0"
  >
    {SOURCE_STANCES.map((stance) => (
      <ToggleGroupItem
        key={stance}
        value={stance}
        className={cn(
          itemClass[size],
          "min-w-0 text-muted-foreground transition-colors duration-150 ease-standard data-[state=on]:bg-accent data-[state=on]:text-foreground",
        )}
      >
        {SOURCE_STANCE_LABEL[stance]}
      </ToggleGroupItem>
    ))}
  </ToggleGroup>
);
