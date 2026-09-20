import { LifeBuoyIcon, LogOutIcon, SettingsIcon, UsersIcon } from "lucide-react";
import { Link, useNavigate } from "react-router";

import { PopoverContent } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { useSessionStore } from "@/stores/sessionStore";

const menuItemClass =
  "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:bg-accent/60 focus-visible:text-foreground";

interface AccountMenuActionsProps {
  canManageMembers: boolean;
  canManageSettings: boolean;
  onClose: () => void;
}

export const AccountMenuActions = ({ canManageMembers, canManageSettings, onClose }: AccountMenuActionsProps) => {
  const navigate = useNavigate();
  const logout = useSessionStore((s) => s.logout);

  const onLogout = () => {
    onClose();
    logout();
    // Replace, not push: the current URL would 404 once the router rebuilds logged-out.
    navigate("/", { replace: true });
  };

  return (
    <PopoverContent side="top" align="start" sideOffset={8} className="w-56 p-1.5">
      <div className="flex flex-col gap-0.5">
        {canManageMembers && (
          <Link to="/members" className={menuItemClass} onClick={onClose}>
            <UsersIcon className="size-4 shrink-0" aria-hidden />
            <span>Members</span>
          </Link>
        )}
        {canManageSettings && (
          <Link to="/settings" className={menuItemClass} onClick={onClose}>
            <SettingsIcon className="size-4 shrink-0" aria-hidden />
            <span>Settings</span>
          </Link>
        )}
        <a
          href="https://github.com/otal-labs/nexul/issues"
          target="_blank"
          rel="noreferrer"
          className={menuItemClass}
          onClick={onClose}
        >
          <LifeBuoyIcon className="size-4 shrink-0" aria-hidden />
          <span>Support</span>
        </a>
        <button
          type="button"
          onClick={onLogout}
          className={cn(
            menuItemClass,
            "text-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:bg-destructive/10 focus-visible:text-destructive",
          )}
        >
          <LogOutIcon className="size-4 shrink-0" aria-hidden />
          <span>Logout</span>
        </button>
      </div>
    </PopoverContent>
  );
};
