import { Search, X } from "lucide-react";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface RoleFindFieldProps {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  label: string;
  className?: string;
}

// The one find field on a role's page and in its pickers: the search mark, the field, and a clear button once it holds text.
export const RoleFindField = ({ value, onChange, placeholder, label, className }: RoleFindFieldProps) => (
  <div className={cn("relative shrink-0", className)}>
    <Search aria-hidden className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
    <Input value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} aria-label={label} className="h-8 pr-8 pl-8" />
    {value && (
      <button
        type="button"
        aria-label="Clear the search"
        onClick={() => onChange("")}
        className="absolute top-1/2 right-1.5 flex size-6 -translate-y-1/2 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X aria-hidden className="size-3.5" />
      </button>
    )}
  </div>
);
