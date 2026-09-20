import { Link } from "react-router";

import { AccountMenu } from "@/components/AccountMenu";
import { ThemeToggle } from "@/components/ThemeToggle";
import { useServerVersion } from "@/hooks/VersionHooks";
import { cn } from "@/lib/utils";

interface SidebarFooterProps {
  collapsed: boolean;
  isLoggedIn: boolean;
}

const INSTANCE_VERSION_SECTION_URL = "/settings?section=instance#instance-version";

export const SidebarFooter = ({ collapsed, isLoggedIn }: SidebarFooterProps) => {
  const server = useServerVersion(isLoggedIn);

  return (
    <div className={cn("border-t border-border", collapsed ? "p-2" : "px-2 py-1")}>
      <div className={cn("flex h-14 items-center gap-1", collapsed && "h-auto flex-col")}>
        {isLoggedIn && (
          <div className={cn("min-w-0", !collapsed && "flex-1")}>
            <AccountMenu collapsed={collapsed} />
          </div>
        )}
        <ThemeToggle />
      </div>
      {!collapsed && server.data && (
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 pb-1.5">
          <span className="font-mono text-[11px] text-muted-foreground">{server.data.version}</span>
          {server.data.update_available && server.data.latest && (
            <Link
              to={INSTANCE_VERSION_SECTION_URL}
              className="inline-flex shrink-0 items-center rounded-full bg-warning/15 px-2 py-0.5 font-mono text-[11px] font-medium text-warning"
            >
              Update available · {server.data.latest.version}
            </Link>
          )}
        </div>
      )}
    </div>
  );
};
