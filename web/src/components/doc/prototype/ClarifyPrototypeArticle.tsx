import { DocBodyView } from "@/components/doc/DocBodyView";
import { DOC_AFTER, DOC_BEFORE } from "@/components/doc/prototype/ClarifyPrototypeData";
import { useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import { cn } from "@/lib/utils";

interface ClarifyPrototypeArticleProps {
  title: string;
  className?: string;
}

// The doc's article card, read-only here: the client's words until a round finds no gaps, then the written doc.
export const ClarifyPrototypeArticle = ({ title, className }: ClarifyPrototypeArticleProps) => {
  const written = useClarifyPrototypeStore((s) => s.written);
  return (
    <article className={cn("relative min-w-0 rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10 lg:p-14", className)}>
      <h1 className="mt-3 text-center text-4xl font-semibold tracking-tight text-balance sm:text-5xl">{title}</h1>
      <div className="mt-7">
        <DocBodyView key={written ? "after" : "before"} body={written ? DOC_AFTER : DOC_BEFORE} />
      </div>
    </article>
  );
};
