import { CircleHelp, X } from "lucide-react";

interface OriginUnknownRowProps {
  onRemove: () => void;
}

// The found-in marker a reporter sets when nobody knows which ticket a bug came from.
export const OriginUnknownRow = ({ onRemove }: OriginUnknownRowProps) => (
  <li className="flex items-center gap-2 px-2 py-1 text-sm text-muted-foreground">
    <CircleHelp className="size-3.5 shrink-0" aria-hidden />
    <span className="flex-1 py-1">Origin unknown</span>
    <button
      type="button"
      aria-label="Remove origin unknown"
      onClick={onRemove}
      className="flex size-7 shrink-0 items-center justify-center rounded-md transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
    >
      <X className="size-3.5" aria-hidden />
    </button>
  </li>
);
