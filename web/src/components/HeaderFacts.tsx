import type { ReactNode } from "react";

// A page header's facts, not a tagline: what the page holds right now, 16px apart on every page.
export const HeaderFacts = ({ parts }: { parts: (ReactNode | false | undefined)[] }) => (
  <span className="inline-flex flex-wrap items-center gap-x-4 gap-y-1">
    {parts.filter(Boolean).map((part, i) => (
      <span key={i}>{part}</span>
    ))}
  </span>
);
