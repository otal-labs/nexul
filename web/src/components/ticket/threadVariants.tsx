import { useSearchParams } from "react-router";

// Prototype only: the three candidate thread columns, picked with ?thread=a|b|c.
export interface ThreadVariant {
  key: "a" | "b" | "c" | "d" | "e" | "stacked";
  label: string;
  grid: string;
  body: string;
  thread: string;
  rail: string;
  frame: string;
  box: string;
  composerTop: boolean;
  empty: "sentence" | "card" | "composer";
  wide?: boolean;
}

export const STACKED: ThreadVariant = {
  key: "stacked",
  label: "Thread under the body",
  grid: "grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem]",
  body: "min-w-0 space-y-8 lg:col-start-1 lg:row-start-1",
  thread: "min-w-0 lg:col-start-1 lg:row-start-2",
  rail: "min-w-0 lg:col-start-2 lg:row-start-1 lg:row-span-2",
  frame: "space-y-3 border-t border-border pt-6",
  box: "h-96",
  composerTop: false,
  empty: "sentence",
};

// D and E fill the page: TicketPage drops its max width and makes itself the @container these sizes read.
const fullWidth = (composerTop: boolean): Omit<ThreadVariant, "key" | "label"> => ({
  grid: "grid gap-8 @min-[46rem]:grid-cols-[clamp(20rem,26cqw,28rem)_minmax(0,1fr)] @min-[70rem]:grid-cols-[clamp(20rem,26cqw,28rem)_minmax(0,1fr)_18rem]",
  body: "min-w-0 space-y-8 @min-[46rem]:col-start-2 @min-[46rem]:row-start-1 [&_.ProseMirror]:mx-auto [&_.ProseMirror]:max-w-[75ch]",
  thread: "min-w-0 @min-[46rem]:col-start-1 @min-[46rem]:row-start-1 @min-[46rem]:row-span-2",
  rail: "min-w-0 @min-[46rem]:col-start-2 @min-[46rem]:row-start-2 @min-[70rem]:col-start-3 @min-[70rem]:row-start-1 @min-[70rem]:row-span-2",
  frame:
    "space-y-3 border-t border-border pt-6 @min-[46rem]:sticky @min-[46rem]:top-4 @min-[46rem]:flex @min-[46rem]:h-[calc(100dvh-2rem)] @min-[46rem]:flex-col @min-[46rem]:border-t-0 @min-[46rem]:pt-0",
  box: "h-96 @min-[46rem]:h-auto @min-[46rem]:min-h-0 @min-[46rem]:flex-1",
  composerTop,
  empty: composerTop ? "composer" : "sentence",
  wide: true,
});

export const THREAD_VARIANTS: Record<"a" | "b" | "c" | "d" | "e", ThreadVariant> = {
  a: {
    key: "a",
    label: "A: slim 18rem column from 1280px, sticky and full screen tall, composer pinned at its foot",
    grid: "grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem] xl:grid-cols-[18rem_minmax(0,1fr)_18rem]",
    body: "min-w-0 space-y-8 lg:col-start-1 lg:row-start-1 xl:col-start-2",
    thread: "min-w-0 lg:col-start-1 lg:row-start-2 xl:row-start-1 xl:row-span-2",
    rail: "min-w-0 lg:col-start-2 lg:row-start-1 lg:row-span-2 xl:col-start-3",
    frame:
      "space-y-3 border-t border-border pt-6 xl:sticky xl:top-4 xl:flex xl:h-[calc(100dvh-2rem)] xl:flex-col xl:border-t-0 xl:pt-0",
    box: "h-96 xl:h-auto xl:min-h-0 xl:flex-1",
    composerTop: false,
    empty: "sentence",
  },
  b: {
    key: "b",
    label: "B: wide 22rem column from 1440px, as tall as the body and scrolling with it, composer at the body's end",
    grid: "grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem] min-[90rem]:grid-cols-[22rem_minmax(0,1fr)_18rem]",
    body: "min-w-0 space-y-8 lg:col-start-1 lg:row-start-1 min-[90rem]:col-start-2",
    thread:
      "min-w-0 lg:col-start-1 lg:row-start-2 min-[90rem]:relative min-[90rem]:row-start-1 min-[90rem]:row-span-2 min-[90rem]:min-h-[32rem]",
    rail: "min-w-0 lg:col-start-2 lg:row-start-1 lg:row-span-2 min-[90rem]:col-start-3",
    frame:
      "space-y-3 border-t border-border pt-6 min-[90rem]:absolute min-[90rem]:inset-0 min-[90rem]:flex min-[90rem]:flex-col min-[90rem]:border-t-0 min-[90rem]:pt-0",
    box: "h-96 min-[90rem]:h-auto min-[90rem]:min-h-0 min-[90rem]:flex-1",
    composerTop: false,
    empty: "card",
  },
  c: {
    key: "c",
    label: "C: 20rem column from 1024px, sticky, composer on top; the rail sits under the body until 1440px",
    grid: "grid gap-8 lg:grid-cols-[20rem_minmax(0,1fr)] min-[90rem]:grid-cols-[20rem_minmax(0,1fr)_18rem]",
    body: "min-w-0 space-y-8 lg:col-start-2 lg:row-start-1",
    thread: "min-w-0 lg:col-start-1 lg:row-start-1 lg:row-span-2",
    rail: "min-w-0 lg:col-start-2 lg:row-start-2 min-[90rem]:col-start-3 min-[90rem]:row-start-1 min-[90rem]:row-span-2",
    frame:
      "space-y-3 border-t border-border pt-6 lg:sticky lg:top-4 lg:flex lg:h-[calc(100dvh-2rem)] lg:flex-col lg:border-t-0 lg:pt-0",
    box: "h-96 lg:h-auto lg:min-h-0 lg:flex-1",
    composerTop: true,
    empty: "composer",
  },
  d: {
    key: "d",
    label: "D: full width, column grows 20 to 28rem, sticky and full screen tall, composer at the foot; rail under the body when narrow",
    ...fullWidth(false),
  },
  e: {
    key: "e",
    label: "E: full width like D, composer on top",
    ...fullWidth(true),
  },
};

export const useThreadVariant = (embedded: boolean): ThreadVariant => {
  const [params] = useSearchParams();
  if (embedded) return STACKED;
  const key = params.get("thread");
  if (key === "b" || key === "c" || key === "d" || key === "e") return THREAD_VARIANTS[key];
  return THREAD_VARIANTS.a;
};
