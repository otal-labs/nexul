import { Check } from "lucide-react";

import { TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PAIRING_STEPS, type PairingStep } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface PairingStepTabsProps {
  step: PairingStep;
  // The furthest step the user may open; everything after it reads as upcoming.
  reachable: PairingStep;
  // The first step the user may open; a dialog opened for a paired computer starts at Set up.
  earliest?: PairingStep;
}

// Segmented step track: done, current, and upcoming read by contrast alone; below 640px it is one line of text.
export const PairingStepTabs = ({ step, reachable, earliest = "connect" }: PairingStepTabsProps) => {
  const current = PAIRING_STEPS.findIndex((s) => s.value === step);
  const furthest = PAIRING_STEPS.findIndex((s) => s.value === reachable);
  const first = PAIRING_STEPS.findIndex((s) => s.value === earliest);
  return (
    <>
      <p className="text-xs text-muted-foreground sm:hidden">
        Step {current + 1} of {PAIRING_STEPS.length} · <span className="text-foreground">{PAIRING_STEPS[current]?.label}</span>
      </p>
      <TabsList className="hidden h-9 w-full rounded-lg border border-border bg-surface-2 p-[3px] sm:flex">
        {PAIRING_STEPS.map((s, i) => (
          <TabsTrigger
            key={s.value}
            value={s.value}
            disabled={i > furthest || i < first}
            className={cn("text-xs", i < current && "text-foreground dark:text-foreground")}
          >
            {i < current && <Check className="size-3.5" aria-hidden />}
            Step {i + 1}: {s.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </>
  );
};
