import type { ReactNode } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface RailTooltipProps {
  label: string;
  // Only the icon rail names its items in a tooltip; the open sidebar shows the label beside the icon.
  collapsed: boolean;
  children: ReactNode;
}

export const RailTooltip = ({ label, collapsed, children }: RailTooltipProps) => (
  <Tooltip {...(collapsed ? {} : { open: false })}>
    <TooltipTrigger asChild>{children}</TooltipTrigger>
    <TooltipContent side="right">{label}</TooltipContent>
  </Tooltip>
);
