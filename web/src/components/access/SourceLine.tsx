import type { PermissionVerdict } from "@/models/Permission";
import { sentenceParts, verdictTone } from "@/models/PermissionPanel";
import { cn } from "@/lib/utils";

const dotClass = {
  allowed: "bg-success",
  denied: "bg-destructive",
  none: "bg-muted-foreground/50",
};

interface SourceLineProps {
  verdict: PermissionVerdict;
  className?: string;
}

// The dot is the answer; the sentence says what decided it, the entity and role it names in medium weight.
export const SourceLine = ({ verdict, className }: SourceLineProps) => (
  <p className={cn("flex min-w-0 items-start gap-2 text-xs text-muted-foreground", className)}>
    <span className={cn("mt-[5px] size-1.5 shrink-0 rounded-full", dotClass[verdictTone(verdict.allowed, verdict.source)])} aria-hidden />
    <span className="min-w-0 text-pretty">
      {sentenceParts(verdict.source).map((part, i) => (
        <span key={i} className={cn(part.strong && "font-medium text-foreground")}>
          {part.text}
        </span>
      ))}
    </span>
  </p>
);
