import { LoadingPlaceholder, PlaceholderBar } from "@/components/LoadingPlaceholder";
import { ticketPageLayout } from "@/components/ticket/ticketPageLayout";
import { cn } from "@/lib/utils";

interface TicketSkeletonProps {
  embedded: boolean;
}

const BODY_LINES = ["w-full", "w-11/12", "w-4/5", "w-full", "w-2/3"];
const RAIL_ROWS = ["w-24", "w-32", "w-28", "w-20", "w-36"];

// The ticket page's own grid while the ticket loads: header, tab line and body on the left, the details rail on the right.
export const TicketSkeleton = ({ embedded }: TicketSkeletonProps) => {
  const layout = ticketPageLayout(embedded);
  return (
    <LoadingPlaceholder label="Loading ticket" className={layout.page} data-pane-layout={embedded ? undefined : ""}>
      <div className={cn("border-b border-border pb-5", layout.header)}>
        <PlaceholderBar className="mb-2 h-4 w-48" />
        <PlaceholderBar className="h-8 w-2/3" />
        <div className="mt-2 flex items-center gap-3">
          <PlaceholderBar className="h-5 w-16" />
          <PlaceholderBar className="h-5 w-24" />
          <PlaceholderBar className="h-5 w-32" />
        </div>
      </div>
      <div className={layout.body}>
        <PlaceholderBar className={cn("mb-2 h-8 w-20", layout.detailsToggle)} />
        <div className={layout.rail}>
          <div className="flex flex-col gap-4 pt-3">
            {RAIL_ROWS.map((width) => (
              <PlaceholderBar key={width} className={cn("h-5", width)} />
            ))}
          </div>
        </div>
        <div className={cn(layout.tabs, "flex flex-col gap-6")}>
          <div className="flex h-9 items-center gap-6 border-b border-border">
            <PlaceholderBar className="h-4 w-12" />
            <PlaceholderBar className="h-4 w-12" />
          </div>
          <div className="flex max-w-[68ch] flex-col gap-3">
            {BODY_LINES.map((width, line) => (
              <PlaceholderBar key={line} className={cn("h-4", width)} />
            ))}
          </div>
        </div>
      </div>
    </LoadingPlaceholder>
  );
};
