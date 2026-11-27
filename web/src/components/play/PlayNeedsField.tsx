import { useId, useState } from "react";
import { XIcon } from "lucide-react";
import { Controller, type Control } from "react-hook-form";

import { LabelChip } from "@/components/LabelChip";
import { Input } from "@/components/ui/input";
import { useListComputers } from "@/hooks/PairingHooks";
import type { SavePlayFormData } from "@/models/Play";

interface PlayNeedsFieldProps {
  control: Control<SavePlayFormData>;
  // Inside a label row that already names it, so the field's own label is only for screen readers.
  bare?: boolean;
}

// The labels a computer must have to take the play's runs (ADR 0197), suggested from the labels on your own computers.
export const PlayNeedsField = ({ control, bare = false }: PlayNeedsFieldProps) => {
  const listId = useId();
  const [draft, setDraft] = useState("");
  const { data: computers } = useListComputers();
  const known = [...new Set((computers ?? []).flatMap((c) => [...(c.labels ?? []), ...(c.derived_labels ?? [])]))].sort();

  return (
    <Controller
      control={control}
      name="needs"
      render={({ field }) => {
        const add = () => {
          const need = draft.trim().toLowerCase();
          setDraft("");
          if (need === "" || field.value.includes(need)) return;
          field.onChange([...field.value, need]);
        };
        return (
          <div className="space-y-2">
            <label htmlFor={`${listId}-input`} className={bare ? "sr-only" : "text-sm font-medium"}>
              Needs
            </label>
            <div className="flex flex-wrap items-center gap-1.5">
              {field.value.map((need) => (
                <LabelChip key={need} label={need} className="pr-1">
                  <button
                    type="button"
                    aria-label={`Remove ${need}`}
                    onClick={() => field.onChange(field.value.filter((n) => n !== need))}
                    className="-my-0.5 ml-0.5 self-center rounded-full p-0.5 text-muted-foreground hover:text-foreground"
                  >
                    <XIcon aria-hidden className="size-3" />
                  </button>
                </LabelChip>
              ))}
              <Input
                id={`${listId}-input`}
                list={listId}
                placeholder="Add a label"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key !== "Enter") return;
                  e.preventDefault();
                  add();
                }}
                className="h-7 w-40 text-xs"
              />
              <datalist id={listId}>
                {known.map((label) => (
                  <option key={label} value={label} />
                ))}
              </datalist>
            </div>
            <p className="text-xs text-muted-foreground">Its runs go only to a computer with every one of these.</p>
          </div>
        );
      }}
    />
  );
};
