import type { ReactNode } from "react";

import { microheaderClass } from "@/components/Microheader";

interface FactProps {
  label: string;
  children: ReactNode;
}

// One cell of a facts grid (design-language "Stack detail page"): a mono microheader over its value, inside a <dl>.
export const Fact = ({ label, children }: FactProps) => (
  <div className="min-w-0">
    <dt className={microheaderClass}>{label}</dt>
    <dd className="mt-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-sm">{children}</dd>
  </div>
);
