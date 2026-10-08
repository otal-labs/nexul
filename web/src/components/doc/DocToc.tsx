import { useEffect, useState } from "react";

import type { DocHeading } from "@/components/doc/docHeadings";
import { cn } from "@/lib/utils";

interface DocTocProps {
  headings: DocHeading[];
}

// Matches by text+level, not anchor ids, since the body view mounts its own Tiptap instance.
function findHeadingElement(heading: DocHeading): HTMLElement | null {
  const els = document.querySelectorAll<HTMLElement>(
    ".doc-body-view h1, .doc-body-view h2, .doc-body-view h3, .doc-body-view h4, .doc-body-view h5, .doc-body-view h6",
  );
  for (const el of els) {
    if (el.textContent === heading.text && el.tagName === `H${heading.level}`) {
      if (!el.dataset.docHeadingId) el.dataset.docHeadingId = heading.id;
      return el;
    }
  }
  return null;
}

// Spy attaches once heading elements exist (MutationObserver) since the editor renders async.
export const DocToc = ({ headings }: DocTocProps) => {
  const [activeId, setActiveId] = useState<string | null>(null);

  useEffect(() => {
    if (headings.length === 0) return;
    let observer: IntersectionObserver | null = null;
    let mutation: MutationObserver | null = null;

    const attach = () => {
      const els = headings
        .map(findHeadingElement)
        .filter((el): el is HTMLElement => el !== null);
      if (els.length === 0) return false;
      if (typeof IntersectionObserver === "undefined") return true;
      observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (entry.isIntersecting && entry.target instanceof HTMLElement) {
              const id = entry.target.dataset.docHeadingId;
              if (id) setActiveId(id);
            }
          }
        },
        { rootMargin: "-20% 0px -70% 0px" },
      );
      for (const el of els) observer.observe(el);
      return true;
    };

    if (!attach()) {
      mutation = new MutationObserver(() => {
        if (attach()) mutation?.disconnect();
      });
      mutation.observe(document.body, { childList: true, subtree: true });
    }
    return () => {
      observer?.disconnect();
      mutation?.disconnect();
    };
  }, [headings]);

  const jump = (heading: DocHeading) => {
    setActiveId(heading.id);
    const el = findHeadingElement(heading);
    if (!el) return;
    const reduceMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
    el.scrollIntoView?.({ behavior: reduceMotion ? "auto" : "smooth", block: "start" });
  };

  if (headings.length === 0) return null;
  return (
    <section aria-label="On this page">
      <h2 className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">
        On this page
      </h2>
      <ul className="mt-2.5 space-y-0.5 border-l border-border">
        {headings.map((heading) => (
          <li key={heading.id}>
            <a
              href={`#${heading.id}`}
              onClick={(event) => {
                event.preventDefault();
                jump(heading);
              }}
              aria-current={activeId === heading.id ? "true" : undefined}
              className={cn(
                "-ml-px block border-l-2 border-transparent py-1 pr-2 text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground",
                heading.level === 1 && "pl-3",
                heading.level === 2 && "pl-5",
                heading.level >= 3 && "pl-7",
                activeId === heading.id && "border-primary font-medium text-primary",
              )}
            >
              {heading.text}
            </a>
          </li>
        ))}
      </ul>
    </section>
  );
};
