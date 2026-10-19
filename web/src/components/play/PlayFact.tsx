import type { ReactNode } from "react";

import { Microheader } from "@/components/Microheader";

interface PlayFactProps {
  label: string;
  children: ReactNode;
}

export const PlayFact = ({ label, children }: PlayFactProps) => (
  <section className="space-y-2">
    <Microheader>{label}</Microheader>
    {children}
  </section>
);
