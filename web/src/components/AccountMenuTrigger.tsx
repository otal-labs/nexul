import { ChevronsUpDownIcon } from "lucide-react";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { PersonAvatar } from "@/components/PersonAvatar";
import { PopoverTrigger } from "@/components/ui/popover";
import { RailTooltip } from "@/components/sidebar/RailTooltip";
import { UpdateDot } from "@/components/UpdateDot";
import { useWelcomeProgress } from "@/hooks/WelcomeHooks";
import { effectiveAvatar, userLabel, loginHandle } from "@/models/User";
import type { User } from "@/models/User";
import { welcomeMenuSummary } from "@/utils/WelcomeMenuUtility";

// Every role held, joined; the line truncates with the rest of the row.
const rolesLine = (role: MyWorkspaceInfo): string => (role.roles ?? []).map((held) => held.name).join(", ") || "No roles";

interface AccountMenuTriggerProps {
  user: User;
  role: MyWorkspaceInfo | undefined;
  collapsed: boolean;
}

export const AccountMenuTrigger = ({ user, role, collapsed }: AccountMenuTriggerProps) => {
  const { data: progress } = useWelcomeProgress();
  const left = progress ? welcomeMenuSummary(progress).left : 0;
  const welcomeLabel = `${left} welcome steps left`;

  return (
  <RailTooltip label={userLabel(user)} collapsed={collapsed}>
    <PopoverTrigger asChild>
      <button
        type="button"
        aria-label={collapsed ? `Account menu for ${userLabel(user)}${left > 0 ? `, ${welcomeLabel}` : ""}` : undefined}
        title={collapsed ? undefined : loginHandle(user.login)}
        className="nav-row flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left hover:bg-accent/60 active:bg-accent data-[state=open]:bg-accent"
      >
        <span className="relative shrink-0">
          <PersonAvatar login={user.login} name={userLabel(user)} src={effectiveAvatar(user)} label="" className="size-8 rounded-lg" />
          {left > 0 && <UpdateDot className="-top-0.5 -right-0.5 bg-destructive" label={collapsed ? undefined : welcomeLabel} />}
        </span>
        {!collapsed && (
          <>
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="truncate text-sm font-medium">{userLabel(user)}</span>
              {role && <span className="truncate text-xs text-muted-foreground">{rolesLine(role)}</span>}
            </span>
            <ChevronsUpDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </>
        )}
      </button>
    </PopoverTrigger>
  </RailTooltip>
  );
};
