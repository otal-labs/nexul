import type { ReactNode } from "react";

import { displayTitleClass } from "@/components/PageHeader";
import { cn } from "@/lib/utils";

interface SignedOutHeaderProps {
  title: string;
  subtitle?: ReactNode;
  // Setup's step segments, above the title.
  progress?: ReactNode;
}

export const SignedOutHeader = ({ title, subtitle, progress }: SignedOutHeaderProps) => (
  <header>
    {progress && <div className="mb-5">{progress}</div>}
    <h1 className={cn(displayTitleClass, "text-[2rem]")}>{title}</h1>
    {subtitle && <p className="mt-3 text-sm text-pretty text-muted-foreground">{subtitle}</p>}
  </header>
);
