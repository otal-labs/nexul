import { ActiveIndicator } from "@/components/ActiveIndicator";
import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import { cn } from "@/lib/utils";

// The sidebar's active row: one highlight per nav that slides to the current page, with the brand edge.
export const SidebarActiveIndicator = () => (
  <ActiveIndicator selector='[aria-current="page"]' className={cn("rounded-md", currentRowFillClass)}>
    <CurrentRowEdge />
  </ActiveIndicator>
);
