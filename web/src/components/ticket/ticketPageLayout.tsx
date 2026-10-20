export interface TicketPageLayout {
  container: string;
  page: string;
  header: string;
  tabs: string;
  ticketTab: string;
  ticketGrid: string;
  article: string;
  thread: string;
  rail: string;
}

const PAGE = "@container/main flex h-full flex-col gap-6 overflow-hidden px-6 pt-6";

// One full-height panel: the header, then the Ticket and Thread tabs. The Ticket tab scrolls under the tab row, and
// its rail joins the body beside it once the panel reaches 52rem; the Thread fills the rest, composer at its foot.
const PANE: TicketPageLayout = {
  container: "h-full max-w-none px-0 sm:px-0",
  page: `panel ${PAGE}`,
  header: "min-w-0 shrink-0",
  tabs: "min-h-0 flex-1",
  ticketTab: "-mx-6 min-h-0 overflow-y-auto px-6 pb-6",
  ticketGrid: "grid content-start gap-8 @min-[52rem]/main:grid-cols-[minmax(0,1fr)_19.5rem] @min-[52rem]/main:gap-x-10",
  article: "min-w-0",
  thread: "flex min-h-0 flex-col pb-4",
  rail: "min-w-0 border-t border-border pt-6 @min-[52rem]/main:border-t-0 @min-[52rem]/main:border-l @min-[52rem]/main:border-border @min-[52rem]/main:pt-0 @min-[52rem]/main:pl-6",
};

// Inside another page (the Inbox) the detail pane is already the panel, so the ticket fills it without a second frame.
const EMBEDDED: TicketPageLayout = { ...PANE, page: PAGE };

export const ticketPageLayout = (embedded: boolean): TicketPageLayout => (embedded ? EMBEDDED : PANE);
