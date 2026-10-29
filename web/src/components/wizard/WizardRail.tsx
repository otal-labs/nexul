import type { ReactNode } from "react";

import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";

interface WizardRailProps {
  ariaLabel: string;
  listLabel?: string;
  label: string;
  // What the wizard is setting up, once it has a name.
  name?: string | null;
  done: number;
  total: number;
  children: ReactNode;
}

// Every wizard's one progress widget: the flow's name, what it sets up, how much is done, then its steps; steps added later rise in.
export const WizardRail = ({ ariaLabel, listLabel, label, name, done, total, children }: WizardRailProps) => (
  <nav aria-label={ariaLabel}>
    <p className={microheaderClass}>{label}</p>
    {name && (
      <p className="mt-1 line-clamp-2 font-mono text-xs break-words text-muted-foreground" title={name}>
        {name}
      </p>
    )}
    <p className="mt-1 font-mono text-xs text-muted-foreground tabular-nums">
      {done} of {total} done
    </p>
    <EnterList as="ol" arrival="rise" aria-label={listLabel} className="-mx-2 mt-5">
      {children}
    </EnterList>
  </nav>
);
