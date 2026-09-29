import type { ReactNode } from "react";

interface ListPaneGroupProps {
  label: string;
  children: ReactNode;
}

export const ListPaneGroup = ({ label, children }: ListPaneGroupProps) => (
  <section aria-label={label}>
    <h2 className="border-b border-border px-3 pt-4 pb-1.5 font-mono text-[11px] font-medium tracking-[0.08em] text-muted-foreground uppercase">
      {label}
    </h2>
    <ul className="divide-y divide-border">{children}</ul>
  </section>
);
