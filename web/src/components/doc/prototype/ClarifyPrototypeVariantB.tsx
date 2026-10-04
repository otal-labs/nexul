import { ChevronDownIcon } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { ClarifyPrototypeArticle } from "@/components/doc/prototype/ClarifyPrototypeArticle";
import { ClarifyPrototypeHeader } from "@/components/doc/prototype/ClarifyPrototypeHeader";
import { ClarifyPrototypeRounds, PANEL_ID } from "@/components/doc/prototype/ClarifyPrototypePanel";
import { ClarifyPrototypeActions, ClarifyPrototypeStatus } from "@/components/doc/prototype/ClarifyPrototypeStatus";
import { useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import type { Doc } from "@/models/Doc";
import { cn } from "@/lib/utils";

// B: a Questions block between the header and the article, folded to its one-line state when nothing is open.
export const ClarifyPrototypeVariantB = ({ doc }: { doc: Doc }) => {
  const { open, setOpen } = useClarifyPrototypeStore(useShallow((s) => ({ open: s.blockOpen, setOpen: s.setBlockOpen })));
  return (
    <div className="mx-auto w-full max-w-4xl">
      <ClarifyPrototypeHeader doc={doc} />
      <section id={PANEL_ID} aria-label="Questions" className="mt-4 rounded-lg border border-border bg-card shadow-card">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2 px-4 py-2.5">
          <button
            type="button"
            aria-expanded={open}
            onClick={() => setOpen(!open)}
            className="group flex min-w-0 flex-1 basis-64 cursor-pointer items-center gap-3 rounded-sm text-left"
          >
            <span className="shrink-0 text-sm font-semibold">Questions</span>
            <ClarifyPrototypeStatus className="flex-1" />
            <ChevronDownIcon
              className={cn(
                "size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground",
                !open && "-rotate-90",
              )}
              aria-hidden
            />
          </button>
          <ClarifyPrototypeActions />
        </div>
        {open && (
          <div className="animate-in fade-in-0 slide-in-from-top-1 border-t border-border px-4 pt-4 pb-5 duration-200 ease-out">
            <ClarifyPrototypeRounds className="max-w-2xl" />
          </div>
        )}
      </section>
      <ClarifyPrototypeArticle title={doc.title} className="mt-4" />
    </div>
  );
};
