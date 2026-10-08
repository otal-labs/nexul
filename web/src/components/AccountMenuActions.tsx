import { LifeBuoyIcon, LogOutIcon } from "lucide-react";

import { PopoverContent } from "@/components/ui/popover";
import { menuItemClass, menuItemDestructiveClass } from "@/components/MenuItem";
import { useLogout } from "@/hooks/AuthHooks";
import { cn } from "@/lib/utils";

interface AccountMenuActionsProps {
  onClose: () => void;
}

export const AccountMenuActions = ({ onClose }: AccountMenuActionsProps) => {
  const logout = useLogout();

  const onLogout = () => {
    onClose();
    logout.mutate();
  };

  return (
    <PopoverContent side="top" align="start" sideOffset={8} className="w-56 p-1">
      <div className="flex flex-col">
        <a
          href="https://github.com/otal-labs/nexul/issues"
          target="_blank"
          rel="noreferrer"
          className={menuItemClass}
          onClick={onClose}
        >
          <LifeBuoyIcon aria-hidden />
          <span>Support</span>
        </a>
        <button
          type="button"
          onClick={onLogout}
          className={cn(menuItemClass, menuItemDestructiveClass)}
        >
          <LogOutIcon aria-hidden />
          <span>Logout</span>
        </button>
      </div>
    </PopoverContent>
  );
};
