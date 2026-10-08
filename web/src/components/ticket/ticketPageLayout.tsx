export interface TicketPageLayout {
  container: string;
  page: string;
  grid: string;
  main: string;
  mainGrid: string;
  body: string;
  thread: string;
  rail: string;
}

// Beside the Inbox list the page is narrow, so the thread stays under the body and the rail waits for the viewport.
const STACKED: TicketPageLayout = {
  container: "p-6",
  page: "",
  grid: "grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem]",
  main: "contents",
  mainGrid: "contents",
  body: "min-w-0 lg:col-start-1 lg:row-start-1",
  thread: "min-w-0 lg:col-start-1 lg:row-start-2",
  rail: "min-w-0 lg:col-start-2 lg:row-start-1 lg:row-span-2",
};

// One panel below 46rem of page width (body, thread, rail in a column); from it the thread and the body are two
// panels, and the rail joins the body's panel beside it once that panel reaches 52rem.
const PANE: TicketPageLayout = {
  container: "h-full max-w-none px-0 sm:px-0",
  page: "@container/page h-full",
  grid: "flex h-full flex-col gap-8 overflow-y-auto p-6 @max-[46rem]/page:panel @min-[46rem]/page:grid @min-[46rem]/page:grid-cols-[clamp(18rem,var(--thread-pane-width,clamp(20rem,26cqw,28rem)),calc(100cqw_-_26rem))_minmax(0,1fr)] @min-[46rem]/page:grid-rows-[minmax(0,1fr)] @min-[46rem]/page:gap-2 @min-[46rem]/page:overflow-visible @min-[46rem]/page:p-0",
  main: "contents @min-[46rem]/page:panel @min-[46rem]/page:@container/main @min-[46rem]/page:col-start-2 @min-[46rem]/page:row-start-1 @min-[46rem]/page:block @min-[46rem]/page:overflow-y-auto",
  mainGrid: "contents @min-[46rem]/page:grid @min-[46rem]/page:gap-8 @min-[46rem]/page:p-6 @min-[52rem]/main:grid-cols-[minmax(0,1fr)_18rem]",
  body: "order-1 min-w-0",
  thread: "order-2 min-w-0 @min-[46rem]/page:panel @min-[46rem]/page:relative @min-[46rem]/page:col-start-1 @min-[46rem]/page:row-start-1 @min-[46rem]/page:flex @min-[46rem]/page:flex-col @min-[46rem]/page:p-4",
  rail: "order-3 min-w-0",
};

export const ticketPageLayout = (embedded: boolean): TicketPageLayout => (embedded ? STACKED : PANE);
