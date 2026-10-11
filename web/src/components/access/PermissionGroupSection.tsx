import type { ReactNode } from "react";

import { Microheader } from "@/components/Microheader";

interface PermissionGroupSectionProps {
  name: string;
  children: ReactNode;
}

// One domain's permissions under its microheader, in the permission-rows box.
export const PermissionGroupSection = ({ name, children }: PermissionGroupSectionProps) => (
  <section className="space-y-2">
    <Microheader>{name}</Microheader>
    <ul className="divide-y divide-border rounded-md border border-input">{children}</ul>
  </section>
);
