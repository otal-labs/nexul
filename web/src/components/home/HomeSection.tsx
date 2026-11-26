import type { ReactNode } from "react";
import { Link } from "react-router";

import { microheaderClass } from "@/components/Microheader";

interface HomeSectionProps {
  title: string;
  /** Where the whole list lives, and the words of the link to it. */
  more: { to: string; label: string };
  children: ReactNode;
}

export const HomeSection = ({ title, more, children }: HomeSectionProps) => (
  <section aria-label={title} className="min-w-0">
    <div className="flex items-baseline justify-between gap-3 pb-1.5">
      <h2 className={microheaderClass}>{title}</h2>
      <Link to={more.to} className="text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
        {more.label}
      </Link>
    </div>
    {children}
  </section>
);
