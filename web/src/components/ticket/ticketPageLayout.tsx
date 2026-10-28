export interface TicketPageLayout {
  container: string;
  page: string;
  header: string;
  body: string;
  detailsToggle: string;
  rail: string;
  railScroll: string;
  railOpen: string;
  tabs: string;
  tabsWide: string;
  ticketTab: string;
  article: string;
  thread: string;
  notesGrid: string;
  notesRail: string;
}

const PAGE = "@container/main flex h-full flex-col gap-6 overflow-hidden px-6 pt-6";

// The header stays put; the tabs and the rail scroll on their own, faded at both ends and padded so nothing fades at rest.
// From a 52rem panel the rail sits beside every tab but Notes; narrower it folds behind a Details row above the tabs.
const PANE: TicketPageLayout = {
  container: "h-full max-w-none px-0 sm:px-0",
  page: `panel ${PAGE}`,
  header: "min-w-0 shrink-0",
  body: "grid min-h-0 flex-1 grid-rows-[auto_minmax(0,max-content)_minmax(10rem,1fr)] @min-[52rem]/main:grid-cols-[minmax(0,1fr)_21rem] @min-[52rem]/main:grid-rows-[minmax(0,1fr)] @min-[52rem]/main:gap-x-10",
  detailsToggle: "@min-[52rem]/main:hidden",
  rail: "hidden min-h-0 min-w-0 @min-[52rem]/main:col-start-2 @min-[52rem]/main:row-start-1 @min-[52rem]/main:block @min-[52rem]/main:border-b-0 @min-[52rem]/main:border-l @min-[52rem]/main:border-border @min-[52rem]/main:pl-6",
  railScroll: "-mx-2 h-full overflow-y-auto pt-3 pb-6 [mask-image:linear-gradient(to_bottom,transparent,#000_12px,#000_calc(100%-24px),transparent)] @min-[52rem]/main:mx-0 @min-[52rem]/main:-mt-3 @min-[52rem]/main:pt-[22.5px]",
  railOpen: "mb-4 block min-h-0 border-b border-border @min-[52rem]/main:mb-0",
  tabs: "min-h-0 min-w-0 @min-[52rem]/main:col-start-1 @min-[52rem]/main:row-start-1",
  tabsWide: "@min-[52rem]/main:col-span-2",
  ticketTab: "-mt-3 min-h-0 overflow-y-auto pt-3 pb-6 [mask-image:linear-gradient(to_bottom,transparent,#000_12px,#000_calc(100%-24px),transparent)]",
  article: "min-w-0",
  thread: "flex min-h-0 flex-col pb-4",
  notesGrid: "grid content-start gap-8 @min-[52rem]/main:grid-cols-[minmax(0,1fr)_21rem] @min-[52rem]/main:gap-x-10",
  notesRail: "min-w-0 border-t border-border pt-6 @min-[52rem]/main:border-t-0 @min-[52rem]/main:border-l @min-[52rem]/main:border-border @min-[52rem]/main:pt-0 @min-[52rem]/main:pl-6",
};

// Inside another page (the Inbox) the detail pane is already the panel, so the ticket fills it without a second frame.
const EMBEDDED: TicketPageLayout = { ...PANE, page: PAGE };

export const ticketPageLayout = (embedded: boolean): TicketPageLayout => (embedded ? EMBEDDED : PANE);
