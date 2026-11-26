import { LoadingPlaceholder, PlaceholderBar } from "@/components/LoadingPlaceholder";
import { cn } from "@/lib/utils";

const WIDTHS = ["w-4/5", "w-3/5", "w-2/3", "w-3/4"];

// Sized like HomeTicketRow: the mark, a title line on the time's baseline, a muted detail line under it.
export const HomeRowsSkeleton = ({ label }: { label: string }) => (
  <LoadingPlaceholder label={label}>
    {WIDTHS.map((width) => (
      <div key={width} className="flex gap-2 py-2">
        <PlaceholderBar className="mt-[3px] size-3.5 rounded-full" />
        <div className="min-w-0 flex-1">
          <div className="flex h-5 items-center gap-3">
            <PlaceholderBar className={cn("h-3.5", width)} />
            <PlaceholderBar className="ml-auto h-3 w-6" />
          </div>
          <PlaceholderBar className="mt-0.5 h-3 w-1/3" />
        </div>
      </div>
    ))}
  </LoadingPlaceholder>
);
