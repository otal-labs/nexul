import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { ContextMenuSub, ContextMenuSubContent, ContextMenuSubTrigger } from "@/components/ui/context-menu";

interface TicketCardMenuSubProps {
  icon: LucideIcon;
  label: string;
  /** The property's value now, muted before the chevron, so the menu reads as the card's properties at a glance. */
  value: string;
  /** How many more values follow the first, kept in view when the first truncates. */
  more?: number;
  children: ReactNode;
  onOpenChange?: (open: boolean) => void;
}

export const TicketCardMenuSub = ({ icon: Icon, label, value, more = 0, children, onOpenChange }: TicketCardMenuSubProps) => (
  <ContextMenuSub {...(onOpenChange ? { onOpenChange } : {})}>
    <ContextMenuSubTrigger>
      <Icon aria-hidden />
      <span className="flex-1">{label}</span>
      <span className="flex max-w-32 min-w-0 gap-1 text-xs text-muted-foreground">
        <span title={value} className="truncate">
          {value}
        </span>
        {more > 0 && <span className="shrink-0">+{more}</span>}
      </span>
    </ContextMenuSubTrigger>
    {/* A long list of people or labels scrolls inside the panel instead of running off the screen. */}
    <ContextMenuSubContent className="max-h-80 w-56 overflow-y-auto">{children}</ContextMenuSubContent>
  </ContextMenuSub>
);
