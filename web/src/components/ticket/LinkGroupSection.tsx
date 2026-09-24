import type { ReactNode } from "react";

interface LinkGroupSectionProps {
  title: string;
  children: ReactNode;
}

// One titled direction inside the linked-tickets section (blocked by, blocks, found in, bugs found in this).
export const LinkGroupSection = ({ title, children }: LinkGroupSectionProps) => (
  <div className="space-y-0.5">
    <h3 className="px-2 text-xs text-muted-foreground">{title}</h3>
    <ul className="flex flex-col">{children}</ul>
  </div>
);
