import { LoadingPlaceholder, PlaceholderBar } from "@/components/LoadingPlaceholder";
import { cn } from "@/lib/utils";

const BODY_LINES = ["w-full", "w-11/12", "w-full", "w-3/4", "w-full", "w-5/6", "w-1/2"];

// The doc page's shape while the doc loads: the crumb line, the title and its meta line, then the body at its reading measure.
export const DocSkeleton = () => (
  <LoadingPlaceholder label="Loading doc" className="mx-auto w-full max-w-5xl">
    {/* Sized like the real actions, so the line wraps under the crumbs in the same narrow panes. */}
    <div className="-mt-6 flex min-h-7 flex-wrap items-center gap-x-4 gap-y-2 pt-6 pb-2">
      <PlaceholderBar className="h-4 w-32" />
      <PlaceholderBar className="ml-auto h-7 w-[25.5rem]" />
    </div>
    <PlaceholderBar className="h-8 w-1/2" />
    <PlaceholderBar className="mt-2 h-5 w-44" />
    <div className="mt-6 flex max-w-[42rem] flex-col gap-3">
      {BODY_LINES.map((width, line) => (
        <PlaceholderBar key={line} className={cn("h-4", width)} />
      ))}
    </div>
  </LoadingPlaceholder>
);
