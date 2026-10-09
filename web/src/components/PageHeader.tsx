import { useState, type ReactNode } from "react";

import { PageBreadcrumb, type Crumb } from "@/components/PageBreadcrumb";
import { cn } from "@/lib/utils";

// The display face for a headline; never below 20px. Empty-state and showcase headlines add their own size.
export const displayTitleClass = "type-display text-balance break-words";

const titleSizeClass = (length: number) => {
  if (length > 100) return "text-xl";
  if (length > 50) return "text-2xl";
  return "text-[1.75rem]";
};

// A title someone typed steps down the scale as it grows (28, 24, 20px), so a sentence-long one stays a header.
export const pageTitleClassFor = (title: string) => `${displayTitleClass} ${titleSizeClass(title.length)}`;

// Marks the box data-clamped while its title runs past it; watching the content too catches a title that grows while clamped.
const markClamped = (el: HTMLDivElement | null) => {
  const content = el?.firstElementChild;
  if (!el || !content) return;
  const mark = () => el.toggleAttribute("data-clamped", el.scrollHeight > el.clientHeight + 1);
  const observer = new ResizeObserver(mark);
  observer.observe(el);
  observer.observe(content);
  return () => observer.disconnect();
};

interface ClampedTitleProps {
  title: string;
  children: ReactNode;
}

// A page title holds three lines; past them its last line fades and a button opens the rest. Editing opens it too.
export const ClampedTitle = ({ title, children }: ClampedTitleProps) => {
  const [open, setOpen] = useState(false);
  return (
    <>
      <div
        ref={markClamped}
        className={cn(
          pageTitleClassFor(title),
          "peer overflow-hidden focus-within:max-h-none data-clamped:not-focus-within:title-clamp-fade",
          !open && "max-h-[3lh]",
        )}
      >
        <div>{children}</div>
      </div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={cn(
          "mt-1 hidden text-xs text-muted-foreground transition-colors duration-150 ease-standard peer-data-clamped:inline-flex hover:text-foreground focus-visible:text-foreground focus-visible:outline-none",
          open && "inline-flex",
        )}
      >
        {open ? "Show less" : "Show full title"}
      </button>
    </>
  );
};

interface PageHeaderProps {
  title: ReactNode;
  crumbs?: Crumb[];
  meta?: ReactNode;
  actions?: ReactNode;
  // A mark before the title and its meta line, centred on the pair: the board's project mark.
  leading?: ReactNode;
  className?: string;
}

export const PageHeader = ({ title, crumbs, meta, actions, leading, className }: PageHeaderProps) => (
  <header className={cn("border-b border-border pb-5", className)}>
    {crumbs && crumbs.length > 0 && <PageBreadcrumb crumbs={crumbs} className="mb-2" />}
    <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
      <div className="flex min-w-0 flex-1 basis-80 items-center gap-3.5">
        {leading}
        <div className="min-w-0 flex-1">
          {typeof title === "string" && (
            <ClampedTitle title={title}>
              <h1 dir="auto" className={pageTitleClassFor(title)}>
                {title}
              </h1>
            </ClampedTitle>
          )}
          {typeof title !== "string" && title}
          {meta && (
            <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
              {meta}
            </div>
          )}
        </div>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </div>
  </header>
);
