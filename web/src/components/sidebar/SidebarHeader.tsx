import { Link } from "react-router";

import { Logo } from "@/components/Logo";
import { CollapseToggleButton } from "@/components/sidebar/CollapseToggleButton";
import { UpdateButton } from "@/components/sidebar/UpdateButton";
import { useServerVersion } from "@/hooks/VersionHooks";
import { cn } from "@/lib/utils";

interface SidebarHeaderProps {
  collapsed: boolean;
  isLoggedIn: boolean;
  onToggleCollapse: () => void;
}

export const SidebarHeader = ({ collapsed, isLoggedIn, onToggleCollapse }: SidebarHeaderProps) => {
  const { data: server } = useServerVersion(isLoggedIn);

  return (
    <>
      <div
        className={cn(
          "relative flex h-14 shrink-0 items-center gap-2.5 pr-2 pl-[18px]",
          collapsed && "justify-center px-0",
        )}
      >
        <Link to="/" className="flex min-w-0 items-center gap-2.5 font-semibold tracking-tight">
          <Logo />
          {!collapsed && <span className="text-[15px]">Nexul</span>}
        </Link>
        {!collapsed && (
          <span
            title={server?.version}
            className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground"
          >
            Beta
          </span>
        )}
        {!collapsed && (
          <div className="ml-auto flex items-center gap-0.5">
            <UpdateButton enabled={isLoggedIn} />
            <CollapseToggleButton collapsed={collapsed} onToggle={onToggleCollapse} />
          </div>
        )}
        {collapsed && <CollapseToggleButton collapsed={collapsed} onToggle={onToggleCollapse} />}
      </div>
      {collapsed && <UpdateButton enabled={isLoggedIn} className="mx-auto mb-1.5" />}
    </>
  );
};
