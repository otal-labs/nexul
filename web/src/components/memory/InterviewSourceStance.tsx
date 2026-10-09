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

// Follow | Question as a segmented pair.
export const InterviewSourceStance = ({ label, value, onChange, disabled = false, size = "xs" }: InterviewSourceStanceProps) => (
  <ToggleGroup
    type="single"
    variant="segmented"
    size="xs"
    aria-label={`Stance for ${label}`}
    disabled={disabled}
    value={value}
    // Clicking the selected side reports it again, so a preselected default can be picked on purpose.
    onValueChange={(next) => onChange((next || value) as SourceStance)}
    className="shrink-0"
  >
    {SOURCE_STANCES.map((stance) => (
      <ToggleGroupItem key={stance} value={stance} className={cn(size === "xs" && "h-6 px-2")}>
        {SOURCE_STANCE_LABEL[stance]}
      </ToggleGroupItem>
    ))}
  </ToggleGroup>
);
