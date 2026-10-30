import { hueStyle, labelDotColor } from "@/components/board/ticketTypeColor";
import { cn } from "@/lib/utils";

interface LabelChipProps extends React.ComponentProps<"span"> {
  label: string;
  /** The label's configured hue name or hex; unset falls back to a hue hashed from the text. */
  color?: string | undefined;
}

// A label is a tag, not a state: a neutral hairline pill with muted text, its own hue only on the leading dot.
export const LabelChip = ({ label, color, className, ...props }: LabelChipProps) => (
  <span
    data-slot="label-chip"
    className={cn(
      "inline-flex max-w-full min-w-0 items-baseline gap-1.5 rounded-full border border-border bg-foreground/[0.04] px-2 py-0.5 text-xs font-medium text-muted-foreground",
      className,
    )}
    {...props}
  >
    <span className={cn("size-1.5 shrink-0 self-center rounded-full", labelDotColor(label, color))} style={hueStyle(color)} aria-hidden />
    <span className="truncate">{label}</span>
  </span>
);
