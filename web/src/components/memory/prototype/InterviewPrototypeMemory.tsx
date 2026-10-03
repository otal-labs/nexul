import { DocBodyView } from "@/components/doc/DocBodyView";
import { MEMORY_BODY } from "@/components/memory/prototype/InterviewPrototypeData";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { cn } from "@/lib/utils";

// The memory as the memory page shows it, read-only and mocked: title, the always-included line, the body.
export const InterviewPrototypeMemory = ({ className }: { className?: string }) => {
  const writing = useInterviewPrototypeStore((s) => s.run === "writing");
  return (
    <article
      aria-busy={writing}
      className={cn(
        "relative rounded-2xl border border-border bg-card p-6 shadow-card transition-opacity duration-200 ease-standard sm:p-10",
        writing && "opacity-50",
        className,
      )}
    >
      <h2 className="mb-2 text-2xl font-semibold tracking-tight">Shopfront interview</h2>
      <p className="mb-6 text-sm text-muted-foreground">Always included in every agent turn in this project; it can't be switched off.</p>
      <DocBodyView body={MEMORY_BODY} />
    </article>
  );
};
