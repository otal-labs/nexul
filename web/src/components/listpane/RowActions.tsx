import { MoreHorizontalIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { RowActionItems } from "@/components/listpane/RowActionItems";
import { dropdownParts, type RowMenu } from "@/components/listpane/rowMenu";

interface RowActionsProps extends RowMenu {
  itemLabel: string;
}

// The row's ⋯, for the keyboard and touch: the same items its right-click menu holds.
export const RowActions = ({ itemLabel, ...menu }: RowActionsProps) => (
  <DropdownMenu>
    {/* The menu trigger wraps the tooltip's, so the button's data-state is the menu's. */}
    <Tooltip>
      <DropdownMenuTrigger asChild>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" className="size-7" aria-label={`More actions for ${itemLabel}`}>
            <MoreHorizontalIcon className="size-3.5" aria-hidden />
          </Button>
        </TooltipTrigger>
      </DropdownMenuTrigger>
      <TooltipContent>More actions</TooltipContent>
    </Tooltip>
    <DropdownMenuContent align="end" className="min-w-36">
      <RowActionItems ui={dropdownParts} {...menu} />
    </DropdownMenuContent>
  </DropdownMenu>
);
