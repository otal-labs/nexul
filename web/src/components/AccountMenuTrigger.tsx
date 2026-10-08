import { ChevronsUpDownIcon } from "lucide-react";

import { PopoverTrigger } from "@/components/ui/popover";
import type { MyWorkspaceInfo } from "@/models/Permission";
import { avatarGradient } from "@/lib/avatarGradient";
import { cn } from "@/lib/utils";
import { effectiveAvatar, userLabel } from "@/models/User";
import type { User } from "@/models/User";

const initials = (name: string) =>
  name
    .split(/[\s@-]/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");

interface AccountMenuTriggerProps {
  user: User;
  role: MyWorkspaceInfo | undefined;
  collapsed: boolean;
}

export const AccountMenuTrigger = ({ user, role, collapsed }: AccountMenuTriggerProps) => (
  <PopoverTrigger asChild>
    <button
      type="button"
      aria-label={collapsed ? `Account menu for ${userLabel(user)}` : undefined}
      title={collapsed ? userLabel(user) : `@${user.login}`}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 focus-visible:ring-[3px] focus-visible:ring-ring/40",
        collapsed && "justify-center",
      )}
    >
      {effectiveAvatar(user) !== "" && (
        <img
          src={effectiveAvatar(user)}
          alt=""
          referrerPolicy="no-referrer"
          className="size-8 shrink-0 rounded-lg object-cover"
        />
      )}
      {effectiveAvatar(user) === "" && (
        <span
          className="flex size-8 shrink-0 items-center justify-center rounded-lg text-xs font-semibold text-white"
          style={{ backgroundImage: avatarGradient(user.login) }}
        >
          {initials(userLabel(user))}
        </span>
      )}
      {!collapsed && (
        <>
          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="truncate text-sm font-medium">{userLabel(user)}</span>
            {role && <span className="truncate text-xs text-muted-foreground">{role.role_name}</span>}
          </span>
          <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </>
      )}
    </button>
  </PopoverTrigger>
);
