import type { ReactNode } from "react";

interface ComputerFactProps {
  label: string;
  children: ReactNode;
}

// One line of a computer row's details: a muted label beside its value, stacked in a narrow row.
export const ComputerFact = ({ label, children }: ComputerFactProps) => (
  <div className="grid gap-x-4 gap-y-1 @md:grid-cols-[7rem_minmax(0,1fr)]">
    <dt className="text-xs text-muted-foreground">{label}</dt>
    <dd className="min-w-0 text-xs">{children}</dd>
  </div>
);
