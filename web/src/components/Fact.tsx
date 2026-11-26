import type { ReactNode } from "react";

import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface FactProps {
  label: string;
  children: ReactNode;
  // A settings card's facts take a muted 12px label, so the value stays the loudest line.
  plain?: boolean;
}

// One cell of a facts grid (design-language "Stack detail page"): a mono microheader over its value, inside a <dl>.
export const Fact = ({ label, children, plain = false }: FactProps) => (
  <div className="min-w-0">
    <dt className={cn(!plain && microheaderClass, plain && "text-xs text-muted-foreground")}>{label}</dt>
    <dd className="mt-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-sm">{children}</dd>
  </div>
);
