import type { ReactNode } from "react";
import { ChevronDownIcon } from "lucide-react";

import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { cn } from "@/lib/utils";

interface InterviewPrototypeSectionProps {
  sectionKey: string;
  label: string;
  meta: string;
  children: ReactNode;
  className?: string;
}

// A collapsible group of question rows with the board swimlane's header: label left, muted count and chevron right.
export const InterviewPrototypeSection = ({ sectionKey, label, meta, children, className }: InterviewPrototypeSectionProps) => {
  const collapsed = useInterviewPrototypeStore((s) => s.collapsed.includes(sectionKey));
  const toggle = useInterviewPrototypeStore((s) => s.toggleSection);

  return (
    <section aria-label={label} className={cn("space-y-2", className)}>
      <h3 className="border-b border-border pb-1.5">
        <button
          type="button"
          aria-expanded={!collapsed}
          onClick={() => toggle(sectionKey)}
          className="group/lane flex w-full cursor-pointer items-center justify-between gap-2 rounded-sm px-0.5 text-left"
        >
          <span className="min-w-0 truncate text-sm font-semibold">{label}</span>
          <span className="flex shrink-0 items-center gap-1 font-mono text-xs text-muted-foreground transition-colors group-hover/lane:text-foreground">
            {meta}
            <ChevronDownIcon
              className={cn("size-3.5 transition-transform duration-150 ease-standard", collapsed && "-rotate-90")}
              aria-hidden
            />
          </span>
        </button>
      </h3>
      <div
        inert={collapsed}
        aria-hidden={collapsed}
        className={cn(
          "grid transition-[grid-template-rows,opacity] motion-reduce:transition-[opacity]",
          collapsed
            ? "[grid-template-rows:0fr] opacity-0 duration-150 ease-standard"
            : "[grid-template-rows:1fr] opacity-100 duration-200 ease-out",
        )}
      >
        <div className="min-h-0 overflow-hidden">{children}</div>
      </div>
    </section>
  );
};
