import { ActiveIndicator } from "@/components/ActiveIndicator";

// The sidebar's active row: one highlight per nav that slides to the current page, with the brand edge.
export const SidebarActiveIndicator = () => (
  <ActiveIndicator selector='[aria-current="page"]' className="rounded-md bg-accent">
    <span className="absolute inset-y-2 left-0 w-0.5 rounded-full bg-brand" />
  </ActiveIndicator>
);
