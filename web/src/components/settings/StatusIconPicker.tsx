import { StatusIcon } from "@/components/board/StatusIcon";
import { cn } from "@/lib/utils";
import { STATUS_ICON_NAMES } from "@/models/Status";

interface StatusIconPickerProps {
  label: string;
  value: string;
  onChange: (icon: string) => void;
}

const optionClass = (active: boolean) =>
  cn(
    "inline-flex size-9 items-center justify-center border-b-2 transition-colors duration-150 ease-standard",
    active
      ? "border-foreground text-foreground"
      : "border-transparent text-muted-foreground hover:text-foreground",
  );

// Plain controlled props (not Controller-bound) so both the create form and row-edit can share it.
// Bare icon targets, no fill/border — selection reads via icon color plus a bottom-border underline.
export const StatusIconPicker = ({ label, value, onChange }: StatusIconPickerProps) => (
  <div role="radiogroup" aria-label={label} className="flex flex-wrap items-center gap-0.5">
    <button
      type="button"
      role="radio"
      aria-checked={value === ""}
      aria-label="No icon"
      className={cn(optionClass(value === ""), "text-[11px] font-medium tracking-wide")}
      onClick={() => onChange("")}
    >
      None
    </button>
    {STATUS_ICON_NAMES.map((icon) => (
      <button
        key={icon}
        type="button"
        role="radio"
        aria-checked={value === icon}
        aria-label={icon}
        className={optionClass(value === icon)}
        onClick={() => onChange(icon)}
      >
        <StatusIcon icon={icon} className="size-4" />
      </button>
    ))}
  </div>
);
