import type { ReactNode } from "react";
import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface SettingsSaveBarProps {
  /** The id of the form this strip saves; the strip sits in the card's footer, outside the form. */
  form: string;
  dirty: boolean;
  saving: boolean;
  /** True for a moment after a save lands (useFlash): Save turns into "Saved". */
  saved: boolean;
  onDiscard: () => void;
  saveLabel?: string;
  /** What saving does, shown while nothing has changed. */
  hint?: ReactNode;
}

const strip = "transition-[opacity,translate,visibility] duration-150 ease-out";
const off = "invisible -translate-x-1 opacity-0";

// The card's footer strip for a form: always there, so nothing below it moves; a change wakes Discard and Save.
export const SettingsSaveBar = ({ form, dirty, saving, saved, onDiscard, saveLabel = "Save", hint }: SettingsSaveBarProps) => (
  <>
    <span className="grid min-w-0 flex-1 text-sm text-muted-foreground">
      <span className={cn("col-start-1 row-start-1 truncate", strip, (dirty || saved || !hint) && off)}>{hint}</span>
      <span className={cn("col-start-1 row-start-1", strip, !dirty && off)}>Unsaved changes</span>
    </span>
    <span className="flex items-center gap-2">
      <Button type="button" variant="ghost" size="sm" className={cn(strip, !dirty && off)} tabIndex={dirty ? 0 : -1} aria-hidden={!dirty} onClick={onDiscard}>
        Discard
      </Button>
      <Button
        type={saved && !dirty ? "button" : "submit"}
        form={form}
        size="sm"
        variant={dirty || saved ? "default" : "outline"}
        loading={saving}
        disabled={!dirty && !saved}
      >
        <span className="swap">
          <span {...(saved ? { "data-off": "" } : {})}>{saveLabel}</span>
          <span {...(saved ? {} : { "data-off": "" })} aria-hidden={!saved} className="inline-flex items-center gap-1.5">
            {saved && <Check className="pop-in size-4" aria-hidden />}
            Saved
          </span>
        </span>
      </Button>
      <span role="status" className="sr-only">
        {saved ? "Saved" : ""}
      </span>
    </span>
  </>
);
