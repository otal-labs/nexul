export interface TicketPageLayout {
  container: string;
  grid: string;
  body: string;
  thread: string;
  rail: string;
}

// Beside the Inbox list the page is narrow, so the thread stays under the body and the rail waits for the viewport.
const STACKED: TicketPageLayout = {
  container: "p-6",
  grid: "grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem]",
  body: "min-w-0 lg:col-start-1 lg:row-start-1",
  thread: "min-w-0 lg:col-start-1 lg:row-start-2",
  rail: "min-w-0 lg:col-start-2 lg:row-start-1 lg:row-span-2",
};

// The page is its own container so the breakpoints follow the width beside the sidebar: the pane from 46rem, the rail from 70rem.
const PANE: TicketPageLayout = {
  container: "p-6 @container max-w-none",
  grid: "grid gap-8 @min-[46rem]:grid-cols-[clamp(18rem,var(--thread-pane-width,clamp(20rem,26cqw,28rem)),calc(100cqw_-_26rem))_minmax(0,1fr)] @min-[70rem]:grid-cols-[clamp(18rem,var(--thread-pane-width,clamp(20rem,26cqw,28rem)),calc(100cqw_-_48rem))_minmax(0,1fr)_18rem]",
  body: "min-w-0 @min-[46rem]:col-start-2 @min-[46rem]:row-start-1",
  thread: "min-w-0 @min-[46rem]:relative @min-[46rem]:col-start-1 @min-[46rem]:row-start-1 @min-[46rem]:row-span-2",
  rail: "min-w-0 @min-[46rem]:col-start-2 @min-[46rem]:row-start-2 @min-[70rem]:col-start-3 @min-[70rem]:row-start-1 @min-[70rem]:row-span-2",
};

export const ticketPageLayout = (embedded: boolean): TicketPageLayout => (embedded ? STACKED : PANE);
