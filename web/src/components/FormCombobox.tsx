import { useMemo, useRef, useState } from "react";
import { CheckIcon, ChevronDownIcon } from "lucide-react";
import {
  Controller,
  useFormState,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

export interface ComboboxOption {
  value: string;
  label: string;
}

interface FormComboboxProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
  label: string;
  options: ComboboxOption[];
  /** Trigger text when nothing's chosen and the clear row's label; picking it stores "" (as FormSelect does). */
  placeholder?: string;
  /** Side-effects on selection (e.g. clearing a dependent field). */
  onChangeValue?: (value: string) => void;
}

// A FormSelect that filters as you type; the list narrows client-side since options are already loaded.
export const FormCombobox = <T extends FieldValues>({
  control,
  name,
  label,
  options,
  placeholder,
  onChangeValue,
}: FormComboboxProps<T>) => {
  const { errors } = useFormState({ control });
  const message = (errors[name]?.message as string | undefined) ?? undefined;
  const labelId = `${name}-label`;
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => o.label.toLowerCase().includes(q));
  }, [options, query]);

  const openWithReset = (next: boolean) => {
    setOpen(next);
    if (next) {
      setQuery("");
      setHighlighted(0);
    }
  };

  const scrollHighlightIntoView = (index: number) => {
    listRef.current?.children[index]?.scrollIntoView({ block: "nearest" });
  };

  return (
    <div className="space-y-2">
      <label id={labelId} className="text-sm font-medium">
        {label}
      </label>
      <Controller
        control={control}
        name={name}
        render={({ field }) => {
          const selected = options.find((o) => o.value === field.value);
          const pick = (value: string) => {
            field.onChange(value);
            onChangeValue?.(value);
            setOpen(false);
          };
          return (
            <Popover open={open} onOpenChange={openWithReset}>
              <PopoverTrigger
                id={name}
                role="combobox"
                aria-expanded={open}
                aria-controls={`${name}-listbox`}
                aria-labelledby={labelId}
                aria-invalid={message != null}
                onBlur={field.onBlur}
                className={cn(
                  "flex h-9 w-full min-w-0 items-center justify-between gap-2 rounded-md border border-input bg-transparent px-3 text-sm whitespace-nowrap shadow-xs outline-none transition-[color,box-shadow,border-color] duration-150 ease-standard hover:border-ring/40 focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/30 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:bg-input/20 [&_svg]:pointer-events-none [&_svg]:shrink-0",
                  selected ? "text-foreground" : "text-muted-foreground",
                )}
              >
                <span className="truncate">{selected?.label ?? placeholder ?? "Select…"}</span>
                <ChevronDownIcon className="size-3.5 text-muted-foreground" aria-hidden />
              </PopoverTrigger>
              <PopoverContent className="w-auto min-w-[var(--radix-popover-trigger-width)] max-w-[var(--radix-popover-content-available-width)] p-1">
                <input
                  // The popover just opened at the user's request; focus belongs in the filter.
                  autoFocus
                  value={query}
                  onChange={(e) => {
                    setQuery(e.target.value);
                    setHighlighted(0);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "ArrowDown") {
                      e.preventDefault();
                      const next = Math.min(highlighted + 1, filtered.length - 1);
                      setHighlighted(next);
                      scrollHighlightIntoView(next);
                    }
                    if (e.key === "ArrowUp") {
                      e.preventDefault();
                      const next = Math.max(highlighted - 1, 0);
                      setHighlighted(next);
                      scrollHighlightIntoView(next);
                    }
                    if (e.key === "Enter") {
                      e.preventDefault();
                      const option = filtered[highlighted];
                      if (option) pick(option.value);
                    }
                  }}
                  placeholder="Type to filter…"
                  aria-label={`Filter ${label.toLowerCase()} options`}
                  className="mb-1 h-8 w-full rounded-sm border border-input bg-transparent px-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring"
                />
                <div id={`${name}-listbox`} role="listbox" ref={listRef} className="max-h-64 overflow-y-auto">
                  {placeholder && !query && (
                    <ComboboxRow selected={!field.value} highlighted={false} onPick={() => pick("")}>
                      {placeholder}
                    </ComboboxRow>
                  )}
                  {filtered.map((option, index) => (
                    <ComboboxRow
                      key={option.value}
                      selected={option.value === field.value}
                      highlighted={index === highlighted}
                      onPick={() => pick(option.value)}
                    >
                      {option.label}
                    </ComboboxRow>
                  ))}
                  {filtered.length === 0 && (
                    <p className="px-2 py-1.5 text-sm text-muted-foreground">No matches.</p>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          );
        }}
      />
      {message && (
        <p
          role="alert"
          className="animate-in fade-in-0 slide-in-from-top-0.5 text-sm text-destructive duration-150 ease-out"
        >
          {message}
        </p>
      )}
    </div>
  );
};

interface ComboboxRowProps {
  selected: boolean;
  highlighted: boolean;
  onPick: () => void;
  children: React.ReactNode;
}

const ComboboxRow = ({ selected, highlighted, onPick, children }: ComboboxRowProps) => (
  <button
    type="button"
    role="option"
    aria-selected={selected}
    // onMouseDown beats the input's blur, keeping click-to-pick reliable.
    onMouseDown={(e) => {
      e.preventDefault();
      onPick();
    }}
    className={cn(
      "relative flex w-full cursor-default items-center gap-2 rounded-sm py-1.5 pr-8 pl-2 text-left text-sm outline-none select-none transition-colors duration-150 ease-standard hover:bg-accent hover:text-accent-foreground",
      highlighted && "bg-accent text-accent-foreground",
    )}
  >
    <span className="truncate">{children}</span>
    {selected && (
      <span className="absolute right-2 flex size-3.5 items-center justify-center">
        <CheckIcon className="size-3.5" aria-hidden />
      </span>
    )}
  </button>
);
