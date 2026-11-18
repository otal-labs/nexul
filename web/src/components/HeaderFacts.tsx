import type { ReactNode } from "react";

// A page header's facts, not a tagline: what the page holds right now, one gap and dot apart on every page.
export const HeaderFacts = ({ parts }: { parts: (ReactNode | false | undefined)[] }) => (
  <span className="inline-flex flex-wrap items-center gap-x-2 gap-y-1">
    {parts.filter(Boolean).map((part, i) => (
      <span key={i} className="inline-flex items-center gap-2">
        {i > 0 && <span aria-hidden>·</span>}
        {part}
      </span>
    ))}
  </span>
);
