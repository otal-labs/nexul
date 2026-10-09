import { AccountMenu } from "@/components/AccountMenu";
import { SettingsGearLink } from "@/components/sidebar/SettingsGearLink";
import { cn } from "@/lib/utils";

interface SidebarFooterProps {
  collapsed: boolean;
}

export const SidebarFooter = ({ collapsed }: SidebarFooterProps) => (
  <div className={cn("border-t border-border", collapsed ? "p-2" : "px-2 py-1")}>
    <div className={cn("flex h-14 items-center gap-1", collapsed && "h-auto flex-col")}>
      <div className={cn("min-w-0", !collapsed && "flex-1")}>
        <AccountMenu collapsed={collapsed} />
      </div>
      <SettingsGearLink collapsed={collapsed} />
    </div>
  </div>
);
