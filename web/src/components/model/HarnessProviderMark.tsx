import { Bot } from "lucide-react";
import type { CSSProperties } from "react";

import { providerMark, type MarkFill } from "@nexul/client-core/providerMarks";

import { cn } from "@/lib/utils";

interface HarnessProviderMarkProps {
  driver: string;
  className?: string;
}

const fillVars = (fill: MarkFill): CSSProperties => ({ "--mark-light": fill.light, "--mark-dark": fill.dark }) as CSSProperties;

const themedFill = "fill-(--mark-light) dark:fill-(--mark-dark)";

// A provider's mark by its driver kind in the brand's own fills; one without a mark gets a generic bot glyph.
export const HarnessProviderMark = ({ driver, className = "size-3.5" }: HarnessProviderMarkProps) => {
  const mark = providerMark(driver);
  if (!mark) return <Bot className={className} aria-hidden />;
  return (
    <svg viewBox={mark.viewBox} className={cn(mark.fill === null ? "fill-current" : themedFill, className)} style={mark.fill ? fillVars(mark.fill) : undefined} aria-hidden>
      {mark.hole && <path d={mark.hole.d} className={themedFill} style={fillVars(mark.hole.fill)} />}
      {mark.paths.map((d) => (
        <path key={d} d={d} {...(mark.evenOdd && { fillRule: "evenodd" as const, clipRule: "evenodd" as const })} />
      ))}
    </svg>
  );
};
