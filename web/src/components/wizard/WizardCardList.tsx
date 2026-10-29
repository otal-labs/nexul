import type { HTMLAttributes } from "react";

import { cn } from "@/lib/utils";

// One wizard list grammar: every row is its own card, the whole card the control, lifted to the opaque card on hover.
export const wizardCardClass =
  "rounded-lg border border-border bg-card/60 transition-colors duration-150 ease-standard hover:bg-card";

export const WizardCardList = ({ className, ...props }: HTMLAttributes<HTMLUListElement>) => (
  <ul className={cn("space-y-2", className)} {...props} />
);

// A choice drawn as a card: the radio 16px in from the card's edge, the picked one ringed in the brand.
export const wizardOptionClass =
  "flex cursor-pointer items-start gap-3 rounded-lg border border-border bg-card/60 px-4 py-3 transition-colors duration-150 ease-standard hover:bg-card has-[[data-state=checked]]:border-brand/70 has-[[data-state=checked]]:bg-card";
