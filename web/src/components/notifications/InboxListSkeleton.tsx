import { LoadingPlaceholder, PlaceholderBar } from "@/components/LoadingPlaceholder";
import { cn } from "@/lib/utils";

const ROWS = ["w-4/5", "w-3/5", "w-11/12", "w-2/3", "w-3/4", "w-1/2"];

// Sized like NotificationItem: the dot column, a title line, then the kind, summary and time line under it.
const InboxRowSkeleton = ({ width }: { width: string }) => (
  <div className="flex items-start px-3 py-2">
    <span className="w-3 shrink-0" />
    <div className="min-w-0 flex-1">
      <PlaceholderBar className={cn("my-[3px] h-3.5", width)} />
      <div className="mt-0.5 flex h-4 items-center gap-1.5">
        <PlaceholderBar className="size-3.5 rounded-full" />
        <PlaceholderBar className="h-3 flex-1" />
        <PlaceholderBar className="h-3 w-8" />
      </div>
    </div>
  </div>
);

export const InboxListSkeleton = () => (
  <LoadingPlaceholder label="Loading notifications">
    {ROWS.map((width) => <InboxRowSkeleton key={width} width={width} />)}
  </LoadingPlaceholder>
);
